package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"regexp"
	"github.com/pkg/errors"
	"fmt"
	"time"
	"golang.org/x/sync/errgroup"
	"math"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"sync"
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
	cw := cloudwatch.New(ses)
	if err := envars.StartGradualRollOut(awsEcs, cw); err != nil {
		log.Fatalf("😭failed roll out new tasks due to: %s", err.Error())
		panic(err)
	}
	log.Infof("🎉service roll out has completed successfully!🎉")
}

func (envars *Envars) CreateNextTaskDefinition(awsEcs ecsiface.ECSAPI) (*ecs.TaskDefinition, error) {
	taskDefinition, err := UnmarshalTaskDefinition(envars.NextTaskDefinitionBase64)
	if err != nil {
		return nil, err
	}
	if out, err := awsEcs.RegisterTaskDefinition(taskDefinition); err != nil {
		return nil, err
	} else {
		return out.TaskDefinition, nil
	}
}

func (envars *Envars) CreateNextService(awsEcs ecsiface.ECSAPI, nextTaskDefinitionArn *string) (*ecs.Service, error) {
	serviceDefinition, err := UnmarshalServiceDefinition(envars.NextServiceDefinitionBase64)
	if err != nil {
		return nil, err
	}
	if out, err := awsEcs.CreateService(serviceDefinition); err != nil {
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

func (envars *Envars) AccumulatePeriodicServiceHealth(
	cw cloudwatchiface.CloudWatchAPI,
	targetGroupArn string,
	startTime time.Time,
	endTime time.Time,
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
	// ロールアウトの検証期間だけ待つ
	timer := time.NewTimer(envars.RollOutPeriod)
	<-timer.C
	getStatics := func(metricName string, unit string) (float64, error) {
		log.Debugf("getStatics: metricName=%s, unit=%s", metricName, unit)
		out, err := cw.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			Namespace:  &nameSpace,
			Dimensions: dimensions,
			MetricName: &metricName,
			StartTime:  &startTime,
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
	requestCnt := 0.0
	elb5xxCnt := 0.0
	target5xxCnt := 0.0
	responseTime := 0.0
	accumulate := func(metricName string, unit string, dest *float64) func() (error) {
		return func() (error) {
			if v, err := getStatics(metricName, unit); err != nil {
				log.Errorf("failed to accumulate CloudWatch's '%s' metric value due to: %s", metricName, err.Error())
				return err
			} else {
				log.Debugf("%s(%s)=%f", metricName, unit, v)
				*dest = v
				return nil
			}
		}
	}
	eg.Go(accumulate("RequestCount", "Sum", &requestCnt))
	eg.Go(accumulate("HTTPCode_ELB_5XX_Count", "Sum", &elb5xxCnt))
	eg.Go(accumulate("HTTPCode_Target_5XX_Count", "Sum", &target5xxCnt))
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

func EstimateRollOutCount(originalTaskCount int, nextDesiredCount int) int {
	ret := 0
	sum := nextDesiredCount
	for ret = 0.0; ; ret += 1.0 {
		add := int(math.Pow(2, float64(ret)))
		if sum+add > originalTaskCount {
			ret++
			break
		}
		sum += add
	}
	return ret
}

func EnsureReplaceCount(rollOutCount int, replacedCount int, originalCount int) int {
	return int(math.Min(
		math.Pow(2, float64(rollOutCount)),
		float64(originalCount-replacedCount)),
	)
}

func (envars *Envars) StartGradualRollOut(awsEcs ecsiface.ECSAPI, cw cloudwatchiface.CloudWatchAPI) (error) {
	// task-definition-nextを作成する
	taskDefinition, err := envars.CreateNextTaskDefinition(awsEcs)
	if err != nil {
		log.Fatalf("😭failed to create new task definition due to: %s", err.Error())
		return err
	}
	// service-nextを作成する
	nextService, err := envars.CreateNextService(awsEcs, taskDefinition.TaskDefinitionArn)
	if err != nil {
		log.Fatalf("😭failed to create new service due to: %s", err.Error())
		return err
	}
	services := []*string{nextService.ServiceName}
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  &envars.Cluster,
		Services: services,
	}); err != nil {
		log.Fatalf("created next service state hasn't reached STABLE state within an interval due to: %s", err.Error())
		return err
	}
	// ロールバックのためのデプロイを始める前の現在のサービスのタスク数
	var originalRunningTaskCount int
	if out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: &envars.Cluster,
		Services: []*string{
			&envars.CurrentServiceName,
		},
	}); err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return err
	} else {
		originalRunningTaskCount = int(*out.Services[0].RunningCount)
	}
	// ロールアウトで置き換えられたタスクの数
	replacedCnt := 0
	// ロールアウト実行回数。CreateServiceで第一陣が配置されてるので1
	rollOutCnt := 1
	// 推定ロールアウト施行回数
	estimatedRollOutCount := EstimateRollOutCount(originalRunningTaskCount, int(*nextService.RunningCount))
	log.Infof(
		"currently %d tasks running on '%s', %d tasks on '%s'. %d times roll out estimated",
		originalRunningTaskCount, envars.CurrentServiceName, *nextService.RunningCount, *nextService.ServiceName, estimatedRollOutCount,
	)
	lb := nextService.LoadBalancers[0]
	// next serviceのperiodic healthが安定し、current serviceのtaskの数が0になるまで繰り返す
	for {
		log.Infof("=== preparing for %dth roll out ===", rollOutCnt)
		if estimatedRollOutCount < rollOutCnt {
			return errors.New(
				fmt.Sprintf(
					"estimated roll out attempts count exceeded: estimated=%d, replaced=%d/%d",
					estimatedRollOutCount, replacedCnt, originalRunningTaskCount,
				),
			)
		}
		startTime := time.Now()
		endTime := startTime
		endTime.Add(envars.RollOutPeriod)
		log.Infof(
			"start accumulating periodic service health of '%s' during %s ~ %s",
			*nextService.ServiceName, startTime.String(), endTime.String(),
		)
		health, err := envars.AccumulatePeriodicServiceHealth(cw, *lb.TargetGroupArn, startTime, endTime)
		if err != nil {
			return err
		}
		log.Infof("periodic health accumulated: availability=%f, responseTime=%f", health.availability, health.responseTime)
		out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster: &envars.Cluster,
			Services: []*string{
				&envars.CurrentServiceName,
				nextService.ServiceName,
			},
		})
		if err != nil {
			log.Errorf("failed to describe next service due to: %s", err.Error())
			return err
		}
		currentService := out.Services[0]
		nextService := out.Services[1]
		if *currentService.RunningCount == 0 && int(*nextService.RunningCount) >= originalRunningTaskCount {
			log.Infof("☀️all tasks successfully have been roll outed!☀️")
			log.Infof("starting to delete old service '%s'", *currentService.ServiceName)
			// すべてのタスクが完全に置き換わったら、current serviceを消す
			if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
				Cluster: &envars.Cluster,
				Service: currentService.ServiceName,
			}); err != nil {
				log.Fatalf("failed to delete empty current service due to: %s", err.Error())
				return err
			}
			log.Infof("'%s' has successfully been deleted. waiting for service state to be INACTIVE", *currentService.ServiceName)
			if err := awsEcs.WaitUntilServicesInactive(&ecs.DescribeServicesInput{
				Cluster:  &envars.Cluster,
				Services: []*string{currentService.ServiceArn},
			}); err != nil {
				log.Errorf("deleted current service state hasn't reached INACTIVE state within an interval due to: %s", err.Error())
				return err
			}
			log.Infof("'%s' is now INACTIVE and will be deleted soon", *currentService.ServiceName)
			return nil
		}
		if envars.AvailabilityThreshold <= health.availability && health.responseTime <= envars.ResponseTimeThreshold {
			// カナリアテストに合格した場合、次のロールアウトに入る
			numToBeReplaced := EnsureReplaceCount(rollOutCnt, replacedCnt, originalRunningTaskCount)
			log.Infof("%dth roll out starting: will replace %d tasks", rollOutCnt, numToBeReplaced)
			replacements, err := envars.RollOut(awsEcs, currentService, nextService, originalRunningTaskCount, rollOutCnt, numToBeReplaced)
			if err != nil {
				log.Fatalf("failed to roll out tasks due to: %s", err.Error())
				return err
			}
			replacedCnt += len(replacements)
			log.Infof(
				"😙 %dth canary test has passed. %d/%d tasks roll outed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
				rollOutCnt, replacedCnt, originalRunningTaskCount,
				health.availability, envars.AvailabilityThreshold, health.responseTime, envars.ResponseTimeThreshold,
			)
			rollOutCnt += 1
		} else {
			// カナリアテストに失敗した場合、task-definition-currentでデプロイを始めた段階のcurrent serviceのタスク数まで戻す
			log.Warnf(
				"😢 %dth canary test has failed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
				rollOutCnt, health.availability, envars.AvailabilityThreshold, health.responseTime, envars.ResponseTimeThreshold,
			)
			return envars.Rollback(awsEcs, currentService, *nextService.ServiceName, originalRunningTaskCount)
		}
	}
}

type TaskReplacement struct {
	oldTask *ecs.Task
	newTask *ecs.Task
}

func (envars *Envars) RollOut(
	awsEcs ecsiface.ECSAPI,
	currentService *ecs.Service,
	nextService *ecs.Service,
	originalTaskCount int,
	rollOutCount int,
	numToBeReplaced int,
) ([]TaskReplacement, error) {
	launchType := "FARGATE"
	desiredStatus := "RUNNING"
	out, err := awsEcs.ListTasks(&ecs.ListTasksInput{
		Cluster:       &envars.Cluster,
		ServiceName:   currentService.ServiceName,
		DesiredStatus: &desiredStatus,
		LaunchType:    &launchType,
	})
	if err != nil {
		log.Errorf("failed to list current tasks due to: %s", err.Error())
		return nil, err
	}
	var replacements []TaskReplacement
	tasks := out.TaskArns
	//TODO: 2018/08/01 ここでRUNNINGタスクの中から止めるものを選択するロジックを考えるべきかもしれない by sakurai
	eg := errgroup.Group{}
	var mux sync.Mutex
	if len(tasks) < numToBeReplaced {
		numToBeReplaced = len(tasks)
	}
	for i := 0; i < numToBeReplaced; i++ {
		task := tasks[i]
		// current-serviceから1つタスクを止めて、next-serviceに1つタスクを追加する
		eg.Go(func() error {
			subEg := errgroup.Group{}
			var (
				oldTask *ecs.Task
				newTask *ecs.Task
			)
			subEg.Go(func() error {
				out, err := awsEcs.StopTask(&ecs.StopTaskInput{
					Cluster: &envars.Cluster,
					Task:    task,
				})
				if err != nil {
					log.Errorf("failed to stop task from current service: taskArn=%s", *task)
					return err
				}
				if err := awsEcs.WaitUntilTasksStopped(&ecs.DescribeTasksInput{
					Cluster: &envars.Cluster,
					Tasks:   []*string{out.Task.TaskArn},
				}); err != nil {
					log.Errorf("stopped current task state hasn't reached STOPPED state within maximum attempt windows: taskArn=%s", out.Task.TaskArn)
					return err
				}
				log.Infof("task '%s' on '%s' has successfully stopped", *task, *currentService.ServiceName)
				oldTask = out.Task
				return nil
			})
			subEg.Go(func() error {
				group := fmt.Sprintf("service:%s", *nextService.ServiceName)
				out, err := awsEcs.StartTask(&ecs.StartTaskInput{
					Cluster:        &envars.Cluster,
					TaskDefinition: nextService.TaskDefinition,
					Group:          &group,
				})
				if err != nil {
					log.Errorf("failed to start task into next service: taskArn=%s", *out.Tasks[0].TaskArn)
					return err
				}
				if err := awsEcs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
					Cluster: &envars.Cluster,
					Tasks:   []*string{out.Tasks[0].TaskArn},
				}); err != nil {
					log.Errorf("launched next task state hasn't reached RUNNING state within maximum attempt windows: taskArn=%s", *out.Tasks[0].TaskArn)
					return err
				}
				newTask = out.Tasks[0]
				log.Infof("task '%s' on '%s has successfully started", *out.Tasks[0].TaskArn, *nextService.ServiceName)
				return nil
			})
			if err := subEg.Wait(); err != nil {
				log.Fatalf("failed to replace tasks due to: %s", err.Error())
				return err
			}
			log.Infof(
				"task replacement (taskArn=%s, service=%s) and (taskArn=%s, service=%s) successfully completed",
				*oldTask.TaskArn, *currentService.ServiceName, *newTask.TaskArn, *nextService.ServiceName,
			)
			mux.Lock()
			replacements = append(replacements, TaskReplacement{
				oldTask: oldTask,
				newTask: newTask,
			})
			mux.Unlock()
			log.Infof("%dth roll out is continuing: replaced %d/%d", rollOutCount, len(replacements), numToBeReplaced)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Fatalf("failed to roll out tasks due to: %s", err.Error())
		return replacements, err
	}
	return replacements, nil
}

func (envars *Envars) Rollback(
	awsEcs ecsiface.ECSAPI,
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
		originalTaskCount, *currentService.RunningCount, rollbackCount,
	)
	rollbackCompletedCount := 0
	rollbackFailedCount := 0
	currentServiceGroup := fmt.Sprintf("service:%s", nextServiceName)
	eg := errgroup.Group{}
	eg.Go(func() error {
		if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
			Cluster: &envars.Cluster,
			Service: &nextServiceName,
		}); err != nil {
			log.Fatalf("failed to delete unhealthy next service due to: %s", err.Error())
			return err
		}
		if err := awsEcs.WaitUntilServicesInactive(&ecs.DescribeServicesInput{
			Cluster:  &envars.Cluster,
			Services: []*string{&nextServiceName},
		}); err != nil {
			log.Fatalf("deleted current service state hasn't reached INACTIVE state within an interval due to: %s", err.Error())
			return err
		}
		return nil
	})
	for i := 0; i < rollbackCount; i++ {
		eg.Go(func() error {
			// タスクを追加
			out, err := awsEcs.StartTask(&ecs.StartTaskInput{
				Cluster:        &envars.Cluster,
				TaskDefinition: currentService.TaskDefinition,
				Group:          &currentServiceGroup,
			})
			if err != nil {
				rollbackFailedCount += 1
				log.Errorf("failed to launch task: taskArn=%s, totalFailure=%d", *out.Tasks[0].TaskArn, rollbackFailedCount)
				return err
			}
			if err := awsEcs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
				Cluster: &envars.Cluster,
				Tasks:   []*string{out.Tasks[0].TaskArn},
			}); err != nil {
				rollbackFailedCount += 1
				log.Errorf("task hasn't reached RUNNING state within maximum attempt windows: taskArn=%s", *out.Tasks[0].TaskArn)
				return err
			}
			rollbackCompletedCount += 1
			log.Infof("rollback is continuing: %d/%d", rollbackCompletedCount, rollbackCount)
			return nil
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
