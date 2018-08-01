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
)

func main() {
	envars, err := EnsureEnvars()
	ses, err := session.NewSession(&aws.Config{
		Region: &envars.Region,
	})
	if err != nil {
		panic(err)
	}
	awsEcs := ecs.New(ses)
	// task-definition-nextを作成する
	if _, err := CreateNextTaskDefinition(envars, awsEcs); err != nil {
		panic(err)
	}
	// service-nextを作成する
	nextService, err := CreateNextService(envars, awsEcs);
	if err != nil {
		panic(err)
	}
	// 漸進的ロールアウト & ロールバックを実行
	if err := StartGradualRollout(envars, ses, awsEcs, nextService); err != nil {
		panic(err)
	}
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
		if out, err := cw.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			Namespace:  &nameSpace,
			Dimensions: dimensions,
			MetricName: &metricName,
			StartTime:  &epoch,
			EndTime:    &endTime,
			Period:     &period,
			Unit:       &unit,
		}); err != nil {
			return 0, nil
		} else {
			var ret float64 = 0
			var err error
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
					ret /= float64(len(out.Datapoints))
				} else {
					err = errors.New("no data points found")
				}
			default:
				err = errors.New(fmt.Sprintf("unsuported unit type: %s", unit))
			}
			return ret, err
		}
	}
	eg := errgroup.Group{}
	var requestCnt = 0.0
	var elb5xxCnt = 0.0
	var target5xxCnt = 0.0
	var responseTime = 0.0
	accumulate := func(metricName string, unit string, dest *float64) func() (error) {
		return func() (error) {
			if v, e := getStatics(metricName, unit); e != nil {
				return e
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

func StartGradualRollout(envars *Envars, ses *session.Session, awsEcs *ecs.ECS, nextService *ecs.Service) (error) {
	// ロールバックのためのデプロイを始める前の現在のサービスのタスク数
	var originalRunningTaskCount int
	if out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: &envars.Cluster,
		Services: []*string{
			&envars.CurrentServiceArn,
		},
	}); err != nil {
		return err
	} else {
		originalRunningTaskCount = int(*out.Services[0].RunningCount)
	}
	// ロールアウトで置き換えられたタスクの数
	replacedCnt := 0
	// ロールアウト実行回数
	rolloutCnt := 0
	// next serviceのperiodic healthが完全に安定し、current serviceの数が0になるまで繰り返す
	for {
		lb := nextService.LoadBalancers[0]
		cw := cloudwatch.New(ses)
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
			return err
		} else {
			currentService := out.Services[0]
			nextService := out.Services[1]
			if *currentService.RunningCount == 0 && int(*nextService.RunningCount) == originalRunningTaskCount {
				// 完全に置き換わった
				return nil
			}
			if envars.AvailabilityThreshold <= envars.AvailabilityThreshold && envars.ResponseTimeThreshold <= health.responseTime {
				// 合格
				err := Rollout(envars, awsEcs, currentService, &replacedCnt, &rolloutCnt)
				if err != nil {
					return err
				}
			} else {
				// カナリアテストに失敗した場合、task-definition-currentでデプロイを始めた段階のcurrent serviceのタスク数まで戻す
				return Rollback(envars, awsEcs, currentService, *nextService.ServiceName, originalRunningTaskCount)
			}
		}
	}
}

func Rollout(
	envars *Envars,
	awsEcs *ecs.ECS,
	currentService *ecs.Service,
	replacedCount *int,
	rolloutCount *int,
) error {
	launchType := "Fargate"
	desiredStatus := "RUNNING"
	if out, err := awsEcs.ListTasks(&ecs.ListTasksInput{
		Cluster:       &envars.Cluster,
		ServiceName:   currentService.ServiceName,
		DesiredStatus: &desiredStatus,
		LaunchType:    &launchType,
	}); err != nil {
		return err
	} else {
		tasks := out.TaskArns
		//TODO: 2018/08/01 ここでRUNNINGタスクの中から止めるものを選択するロジックを考えるべきかもしれない by sakurai
		numToBeReplaced := int(math.Exp2(float64(*rolloutCount)))
		for i := 0; i < numToBeReplaced && len(tasks) > 0; i++ {
			task := tasks[0]
			tasks = tasks[1:]
			if _, err := awsEcs.StopTask(&ecs.StopTaskInput{
				Cluster: &envars.Cluster,
				Task:    task,
			}); err != nil {
				return err
			} else {
				*replacedCount += 1
			}
		}
		*rolloutCount += 1
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
	group := fmt.Sprintf("service:%s", *currentService.ServiceName)
	if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: &envars.Cluster,
		Service: &nextServiceName,
	}); err != nil {
		return err
	}
	eg := errgroup.Group{}
	for i := 0; i < rollbackCount; i++ {
		eg.Go(func() error {
			_, err := awsEcs.StartTask(&ecs.StartTaskInput{
				Cluster:        &envars.Cluster,
				TaskDefinition: currentService.TaskDefinition,
				Group:          &group,
			})
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		// ここに来たらヤバイ
		return err
	}
	return nil
}
