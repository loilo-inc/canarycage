package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"encoding/base64"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"regexp"
	"github.com/pkg/errors"
	"fmt"
	"time"
	"golang.org/x/sync/errgroup"
	"math"
	"github.com/apex/log"
)

func main() {
	envars, err := EnsureEnvars()
	ses, err := session.NewSession(&aws.Config{
		Region: &envars.Region,
	})
	if err != nil {
		log.Fatalf("failed to create new AWS session due to: %s", err.Error())
		panic(err)
	}
	awsEcs := ecs.New(ses)
	// 漸進的ロールアウト & ロールバックを実行
	if err := StartGradualRollOut(envars, ses, awsEcs); err != nil {
		log.Fatalf("😭failed roll out new tasks due to: %s", err.Error())
		panic(err)
	}
	log.Infof("🎉service roll out has completed successfully!🎉")
}

func CreateNextTaskDefinition(envars *Envars, awsEcs *ecs.ECS) (*ecs.TaskDefinition, error) {
	taskDefinitionJson, _ := base64.StdEncoding.DecodeString(envars.NextTaskDefinitionBase64)
	var taskDefinition ecs.RegisterTaskDefinitionInput
	if err := json.Unmarshal([]byte(taskDefinitionJson), &taskDefinition); err != nil {
		return nil, err
	}
	if out, err := awsEcs.RegisterTaskDefinition(&taskDefinition); err != nil {
		return nil, err
	} else {
		return out.TaskDefinition, nil
	}
}

func CreateNextService(envars *Envars, awsEcs *ecs.ECS) (*ecs.Service, error) {
	serviceDefinitionJson, _ := base64.StdEncoding.DecodeString(envars.NextServiceDefinitionBase64)
	var serviceDefinition ecs.CreateServiceInput
	if err := json.Unmarshal([]byte(serviceDefinitionJson), &serviceDefinition); err != nil {
		return nil, err
	}
	if out, err := awsEcs.CreateService(&serviceDefinition); err != nil {
		return nil, err
	} else {
		return out.Service, nil
	}
}

func ExtractAlbId(arn string) (string, error) {
	regex := regexp.MustCompile(`^.+(app/.+?)$`)
	if group := regex.FindStringSubmatch(arn); group == nil || len(group) == 1 {
		return "", errors.New(fmt.Sprintf("could not find alb id in '%s'", arn))
	} else {
		return group[1], nil
	}
}

func ExtractTargetGroupId(arn string) (string, error) {
	regex := regexp.MustCompile(`^.+(targetgroup/.+?)$`)
	if group := regex.FindStringSubmatch(arn); group == nil || len(group) == 1 {
		return "", errors.New(fmt.Sprintf("could not find target group id in '%s'", arn))
	} else {
		return group[1], nil
	}
}

type ServiceHealth struct {
	availability float64
	responseTime float64
}

func AccumulatePeriodicServiceHealth(
	cw *cloudwatch.CloudWatch,
	envars *Envars,
	targetGroupArn string,
	epoch time.Time,
) (*ServiceHealth, error) {
	lbArn := envars.LoadBalancerArn
	tgArn := targetGroupArn
	lbKey := "LoadBalancer"
	lbId, _ := ExtractAlbId(lbArn)
	tgKey := "TargetGroup"
	tgId, _ := ExtractTargetGroupId(tgArn)
	nameSpace := "ApplicationELB"
	period := int64(envars.RollOutPeriod.Seconds())
	dimensions := []*cloudwatch.Dimension{
		{
			Name:  &lbKey,
			Value: &lbId,
		}, {
			Name:  &tgKey,
			Value: &tgId,
		},
	}
	endTime := epoch
	endTime.Add(envars.RollOutPeriod)
	timer := time.NewTimer(envars.RollOutPeriod)
	<-timer.C
	getStatics := func(metricName string, unit string) (float64, error) {
		out, err := cw.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			Namespace:  &nameSpace,
			Dimensions: dimensions,
			MetricName: &metricName,
			StartTime:  &epoch,
			EndTime:    &endTime,
			Period:     &period,
			Unit:       &unit,
		})
		if err != nil {
			log.Fatalf("failed to get CloudWatch's '%s' metric statistics due to: %s", metricName, err.Error())
			return 0, err
		}
		var ret float64 = 0
		switch unit {
		case "Sum":
			for _, v := range out.Datapoints {
				ret += *v.Sum
			}
		case "Average":
			for _, v := range out.Datapoints {
				ret += *v.Average
			}
			if l := len(out.Datapoints); l > 0 {
				ret /= float64(l)
			} else {
				err = errors.New("no data points found")
			}
		default:
			err = errors.New(fmt.Sprintf("unsuported unit type: %s", unit))
		}
		return ret, err

	}
	eg := errgroup.Group{}
	var requestCnt = 0.0
	var elb5xxCnt = 0.0
	var target5xxCnt = 0.0
	var responseTime = 0.0
	accumulate := func(metricName string, unit string, dest *float64) func() (error) {
		return func() (error) {
			if v, err := getStatics(metricName, unit); err != nil {
				log.Errorf("failed to accumulate CloudWatch's '%s' metric value due to: %s", metricName, err.Error())
				return err
			} else {
				*dest = v
				return nil
			}
		}
	}
	eg.Go(accumulate("RequestCount", "Sum", &requestCnt))
	eg.Go(accumulate("ELB5xxCount", "Sum", &elb5xxCnt))
	eg.Go(accumulate("Target5xxCount", "Sum", &target5xxCnt))
	eg.Go(accumulate("TargetResponseTime", "Average", &responseTime))
	if err := eg.Wait(); err != nil {
		log.Errorf("failed to accumulate periodic service health due to: %s", err.Error())
		return nil, err
	} else {
		if requestCnt == 0 && elb5xxCnt == 0 {
			return nil, errors.New("failed to get precise metric data")
		} else {
			avl := (requestCnt - target5xxCnt) / (requestCnt + elb5xxCnt)
			avl = math.Max(0, math.Min(1, avl))
			return &ServiceHealth{
				availability: avl,
				responseTime: responseTime,
			}, nil
		}
	}
}

func StartGradualRollOut(envars *Envars, ses *session.Session, awsEcs *ecs.ECS) (error) {
	// task-definition-nextを作成する
	_, err := CreateNextTaskDefinition(envars, awsEcs)
	if err != nil {
		log.Fatalf("😭failed to create new task definition due to: %s", err.Error())
		return err
	}
	// service-nextを作成する
	nextService, err := CreateNextService(envars, awsEcs)
	if err != nil {
		log.Fatalf("😭failed to create new service due to: %s", err.Error())
		return err
	}
	// ロールバックのためのデプロイを始める前の現在のサービスのタスク数
	var originalRunningTaskCount int
	if out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: &envars.Cluster,
		Services: []*string{
			&envars.CurrentServiceArn,
		},
	}); err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return err
	} else {
		originalRunningTaskCount = int(*out.Services[0].RunningCount)
	}
	// ロールアウトで置き換えられたタスクの数
	replacedCnt := 0
	// ロールアウト実行回数
	rollOutCnt := 0
	lb := nextService.LoadBalancers[0]
	cw := cloudwatch.New(ses)
	// next serviceのperiodic healthが完全に安定し、current serviceの数が0になるまで繰り返す
	for {
		epoch := time.Now()
		var health *ServiceHealth
		var err error
		if health, err = AccumulatePeriodicServiceHealth(cw, envars, *lb.TargetGroupArn, epoch); err != nil {
			return err
		}
		if out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster: &envars.Cluster,
			Services: []*string{
				&envars.CurrentServiceArn,
				nextService.ServiceArn,
			},
		}); err != nil {
			log.Errorf("failed to describe next service due to: %s", err.Error())
			return err
		} else {
			currentService := out.Services[0]
			nextService := out.Services[1]
			if *currentService.RunningCount == 0 && int(*nextService.RunningCount) == originalRunningTaskCount {
				// 完全に置き換わった
				if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
					Cluster: &envars.Cluster,
					Service: currentService.ServiceName,
				}); err != nil {
					log.Fatalf("failed to delete empty current service due to: %s", err.Error())
					return err
				}
				log.Infof("all current tasks have been replaced into next tasks")
				return nil
			}
			if health.availability <= envars.AvailabilityThreshold && envars.ResponseTimeThreshold <= health.responseTime {
				// カナリアテストに合格した場合、次のロールアウトに入る
				if err := RollOut(envars, awsEcs, currentService, nextService, &replacedCnt, &rollOutCnt); err != nil {
					log.Fatalf("failed to roll out tasks due to: %s", err.Error())
					return err
				}
				log.Infof(
					"😙 %dth canary test has passed. %d/%d tasks roll outed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
					rollOutCnt, replacedCnt, originalRunningTaskCount,
					health.availability, envars.AvailabilityThreshold, health.responseTime, envars.ResponseTimeThreshold,
				)
			} else {
				// カナリアテストに失敗した場合、task-definition-currentでデプロイを始めた段階のcurrent serviceのタスク数まで戻す
				log.Warnf(
					"😢 %dth canary test haven't passed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
					rollOutCnt, health.availability, envars.AvailabilityThreshold, health.responseTime, envars.ResponseTimeThreshold,
				)
				return Rollback(envars, awsEcs, currentService, *nextService.ServiceName, originalRunningTaskCount)
			}
		}
	}
}

func RollOut(
	envars *Envars,
	awsEcs *ecs.ECS,
	currentService *ecs.Service,
	nextService *ecs.Service,
	replacedCount *int,
	rollOutCount *int,
) error {
	launchType := "Fargate"
	desiredStatus := "RUNNING"
	if out, err := awsEcs.ListTasks(&ecs.ListTasksInput{
		Cluster:       &envars.Cluster,
		ServiceName:   currentService.ServiceName,
		DesiredStatus: &desiredStatus,
		LaunchType:    &launchType,
	}); err != nil {
		log.Errorf("failed to list current tasks due to: %s", err.Error())
		return err
	} else {
		tasks := out.TaskArns
		//TODO: 2018/08/01 ここでRUNNINGタスクの中から止めるものを選択するロジックを考えるべきかもしれない by sakurai
		numToBeReplaced := int(math.Exp2(float64(*rollOutCount)))
		eg := errgroup.Group{}
		for i := 0; i < numToBeReplaced && len(tasks) > 0; i++ {
			task := tasks[0]
			tasks = tasks[1:]
			// current-serviceから1つタスクを止めて、next-serviceに1つタスクを追加する
			eg.Go(func() error {
				subEg := errgroup.Group{}
				subEg.Go(func() error {
					if _, err := awsEcs.StopTask(&ecs.StopTaskInput{
						Cluster: &envars.Cluster,
						Task:    task,
					}); err != nil {
						log.Errorf("failed to stop task from current service: taskArn=%s", *task)
						return err
					}
					return nil
				})
				subEg.Go(func() error {
					group := fmt.Sprintf("service:%s", *nextService.ServiceName)
					if out, err := awsEcs.StartTask(&ecs.StartTaskInput{
						Cluster:        &envars.Cluster,
						TaskDefinition: nextService.TaskDefinition,
						Group:          &group,
					}); err != nil {
						log.Errorf("failed to start task into next service: taskArn=%s", out.Tasks[0].TaskArn)
						return err
					}
					return nil
				})
				if err := subEg.Wait(); err != nil {
					log.Fatalf("failed to replace tasks due to: %s", err.Error())
					return err
				}
				*replacedCount += 1
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			log.Fatalf("failed to roll out tasks due to: %s", err.Error())
			return err
		}
		*rollOutCount += 1
		return nil
	}
}

func Rollback(
	envars *Envars,
	awsEcs *ecs.ECS,
	currentService *ecs.Service,
	nextServiceName string,
	originalTaskCount int,
) error {
	currentTaskCount := int(*currentService.RunningCount)
	rollbackCount := originalTaskCount - currentTaskCount
	if rollbackCount < 0 {
		rollbackCount = currentTaskCount
	}
	log.Infof(
		"start rollback of current service: originalTaskCount=%d, currentTaskCount=%d, tasksToBeRollback=%d",
		originalTaskCount, currentService.RunningCount, rollbackCount,
	)
	group := fmt.Sprintf("service:%s", *currentService.ServiceName)
	if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: &envars.Cluster,
		Service: &nextServiceName,
	}); err != nil {
		log.Fatalf("failed to delete unhealthy next service due to: %s", err.Error())
		return err
	}
	eg := errgroup.Group{}
	rollbackCompletedCount := 0
	rollbackFailedCount := 0
	iconFunc := RunningIcon()
	for i := 0; i < rollbackCount; i++ {
		eg.Go(func() error {
			// タスクを追加
			out, err := awsEcs.StartTask(&ecs.StartTaskInput{
				Cluster:        &envars.Cluster,
				TaskDefinition: currentService.TaskDefinition,
				Group:          &group,
			})
			if err != nil {
				rollbackFailedCount += 1
				log.Errorf("failed to launch task: taskArn=%s, totalFailure=%d", out.Tasks[0].TaskArn, rollbackFailedCount)
				return err
			}
			if err := awsEcs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
				Cluster: &envars.Cluster,
				Tasks:   []*string{out.Tasks[0].TaskArn},
			}); err != nil {
				rollbackFailedCount += 1
				log.Errorf("task hasn't reached RUNNING state within maximum attempt windows: taskArn=%s", out.Tasks[0].TaskArn)
				return err
			}
			rollbackCompletedCount += 1
			log.Infof("%s️ rollback is continuing: %d/%d", iconFunc(), rollbackCompletedCount, rollbackCount)
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		//TODO: ここに来たらヤバイので手動ロールバックへの動線を貼る
		log.Fatalf(
			"😱service rollback hasn't completed: succeeded=%d/%d, failed=%d",
			rollbackCompletedCount, rollbackCount, rollbackFailedCount,
		)
		return err
	}
	log.Info("😓service rollback has completed successfully")
	return nil
}
