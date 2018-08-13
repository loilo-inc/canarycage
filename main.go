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

func EnsureReplaceCount(
	nextDesiredCount int64,
	totalReplacedCount int,
	totalRollOutCount int,
	originalCount int,
) (addCnt int, removeCnt int) {
	// DesiredCount以下のカナリア追加は意味がないので2回目以降はこの指数より上を使う
	min := int(math.Floor(math.Log2(float64(nextDesiredCount))))
	a := int(math.Min(
		math.Pow(2, float64(min+totalRollOutCount)),
		float64(originalCount-totalReplacedCount)),
	)
	var (
		add    = a
		remove = a
	)
	if totalRollOutCount == 0 {
		add = 0
		remove = int(nextDesiredCount)
	}
	return add, remove
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
	totalReplacedCnt := 0
	// ロールアウト実行回数。CreateServiceで第一陣が配置されてるので1
	totalRollOutCnt := 0
	// 推定ロールアウト施行回数
	estimatedRollOutCount := EstimateRollOutCount(originalRunningTaskCount, int(*nextService.RunningCount))
	log.Infof(
		"currently %d tasks running on '%s', %d tasks on '%s'. %d times roll out estimated",
		originalRunningTaskCount, envars.CurrentServiceName, *nextService.RunningCount, *nextService.ServiceName, estimatedRollOutCount,
	)
	lb := nextService.LoadBalancers[0]
	// next serviceのperiodic healthが安定し、current serviceのtaskの数が0になるまで繰り返す
	for {
		log.Infof("=== preparing for %dth roll out ===", totalRollOutCnt)
		if estimatedRollOutCount <= totalRollOutCnt {
			return errors.New(
				fmt.Sprintf(
					"estimated roll out attempts count exceeded: estimated=%d, rollouted=%d, replaced=%d/%d",
					estimatedRollOutCount, totalRollOutCnt, totalReplacedCnt, originalRunningTaskCount,
				),
			)
		}
		startTime := time.Now()
		endTime := startTime
		endTime.Add(envars.RollOutPeriod)
		addCnt, removeCnt := EnsureReplaceCount(*nextService.DesiredCount, totalRollOutCnt, totalReplacedCnt, originalRunningTaskCount)
		// Phase1: service-nextにtask-nextを指定数配置
		log.Infof("start adding of next tasks. this will add %d tasks to %s", addCnt, *nextService.ServiceName)
		_, err := envars.StartTasks(awsEcs, nextService.ServiceName, nextService.TaskDefinition, addCnt)
		if err != nil {
			log.Fatalf("failed to add next tasks due to: %s", err)
			return err
		}
		log.Infof(
			"start accumulating periodic service health of '%s' during %s ~ %s",
			*nextService.ServiceName, startTime.String(), endTime.String(),
		)
		// Phase2: service-nextのperiodic healthを計測
		health, err := envars.AccumulatePeriodicServiceHealth(cw, *lb.TargetGroupArn, startTime, endTime)
		if err != nil {
			return err
		}
		log.Infof("periodic health accumulated: availability=%f, responseTime=%f", health.availability, health.responseTime)
		if envars.AvailabilityThreshold > health.availability || health.responseTime > envars.ResponseTimeThreshold {
			// カナリアテストに失敗した場合、task-definition-currentでデプロイを始めた段階のcurrent serviceのタスク数まで戻す
			log.Warnf(
				"😢 %dth canary test has failed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
				totalRollOutCnt, health.availability, envars.AvailabilityThreshold, health.responseTime, envars.ResponseTimeThreshold,
			)
			return envars.Rollback(awsEcs, *nextService.ServiceName, originalRunningTaskCount)
		}
		log.Infof(
			"😙 %dth canary test has passed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
			totalRollOutCnt, health.availability, envars.AvailabilityThreshold, health.responseTime, envars.ResponseTimeThreshold,
		)
		// Phase3: service-currentからタスクを指定数消す
		log.Infof(
			"%dth roll out starting: will add %d tasks to '%s' and remove %d tasks from '%s'",
			totalRollOutCnt, addCnt, *nextService.ServiceName, removeCnt, &envars.CurrentServiceName,
		)
		removed, err := envars.StopTasks(awsEcs, &envars.CurrentServiceName, removeCnt)
		if err != nil {
			log.Fatalf("failed to roll out tasks due to: %s", err.Error())
			return err
		}
		totalReplacedCnt += len(removed)
		log.Infof(
			"%dth roll out successfully completed. %d/%d tasks roll outed",
			totalRollOutCnt, totalReplacedCnt, originalRunningTaskCount,
		)
		totalRollOutCnt += 1
		// Phase4: ロールアウトが終わったかどうかを確認
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
		} else {
			log.Infof(
				"roll out is continuing. %d tasks running on '%s', %d tasks on '%s'",
				*currentService.RunningCount, *currentService.ServiceName,
				*nextService.RunningCount, *nextService.ServiceName,
			)
		}
	}
}

func (envars *Envars) StartTasks(
	awsEcs ecsiface.ECSAPI,
	serviceName *string,
	taskDefinition *string,
	numToAdd int,
) ([]*ecs.Task, error) {
	eg := errgroup.Group{}
	var mux sync.Mutex
	var ret []*ecs.Task
	for i := 0; i < numToAdd; i++ {
		eg.Go(func() error {
			group := fmt.Sprintf("service:%s", *serviceName)
			out, err := awsEcs.StartTask(&ecs.StartTaskInput{
				Cluster:        &envars.Cluster,
				TaskDefinition: taskDefinition,
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
			mux.Lock()
			defer mux.Unlock()
			ret = append(ret, out.Tasks...)
			log.Infof("task '%s' on '%s has successfully started", *out.Tasks[0].TaskArn, *serviceName)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Fatalf("registering of next tasks hasn't completed due to: %s", err)
		return nil, err
	}
	log.Infof("registering of next tasks has successfully completed")
	return ret, nil
}

func (envars *Envars) StopTasks(
	awsEcs ecsiface.ECSAPI,
	serviceName *string,
	numToRemove int,
) ([]*ecs.Task, error) {
	out, err := awsEcs.ListTasks(&ecs.ListTasksInput{
		Cluster:       &envars.Cluster,
		ServiceName:   serviceName,
		DesiredStatus: aws.String("RUNNING"),
	})
	if err != nil {
		log.Errorf("failed to list current tasks due to: %s", err.Error())
		return nil, err
	}
	eg := errgroup.Group{}
	var ret []*ecs.Task
	var mux sync.Mutex
	for i := 0; i < numToRemove; i++ {
		task := out.TaskArns[i]
		// current-serviceから1つタスクを止めて、next-serviceに1つタスクを追加する
		eg.Go(func() error {
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
			log.Infof("task '%s' on '%s' has successfully stopped", *task, serviceName)
			mux.Lock()
			defer mux.Unlock()
			ret = append(ret, out.Task)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Fatalf("%dth draining of current tasks hasn't completed due to: %s", err.Error())
		return nil, err
	}
	return ret, nil
}

func (envars *Envars) Rollback(
	awsEcs ecsiface.ECSAPI,
	nextServiceName string,
	originalTaskCount int,
) error {
	out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  &envars.Cluster,
		Services: []*string{aws.String(envars.CurrentServiceName)},
	})
	if err != nil {
		log.Fatalf("failed to describe services due to: %s", err)
		return err
	}
	currentService := out.Services[0]
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
	currentServiceGroup := fmt.Sprintf("service:%s", *currentService.ServiceName)
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
