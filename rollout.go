package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"time"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/apex/log"
	"golang.org/x/sync/errgroup"
	"math"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"sync"
	"errors"
)

type ServiceHealth struct {
	availability float64
	responseTime float64
}

func (envars *Envars) AccumulatePeriodicServiceHealth(
	cw cloudwatchiface.CloudWatchAPI,
	loadBalancerArn string,
	targetGroupArn string,
	startTime time.Time,
	endTime time.Time,
) (*ServiceHealth, error) {
	lbKey := "LoadBalancer"
	lbId, _ := ExtractAlbId(loadBalancerArn)
	tgKey := "TargetGroup"
	tgId, _ := ExtractTargetGroupId(targetGroupArn)
	nameSpace := "ApplicationELB"
	period := envars.RollOutPeriod
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
	timer := time.NewTimer(time.Duration(*envars.RollOutPeriod) * time.Second)
	<-timer.C
	getStatics := func(metricName string, unit string) (float64, error) {
		log.Debugf("getStatics: metricName=%s, unit=%s", metricName, unit)
		out, err := cw.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			Namespace:  &nameSpace,
			Dimensions: dimensions,
			MetricName: &metricName,
			StartTime:  &startTime,
			EndTime:    &endTime,
			Period:     period,
			Unit:       &unit,
		})
		if err != nil {
			log.Errorf("failed to get CloudWatch's '%s' metric statistics due to: %s", metricName, err.Error())
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

func (envars *Envars) StartGradualRollOut(awsEcs ecsiface.ECSAPI, cw cloudwatchiface.CloudWatchAPI) (error) {
	// ロールバックのためのデプロイを始める前の現在のサービスのタスク数
	var originalRunningTaskCount int
	out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: envars.Cluster,
		Services: []*string{
			envars.ServiceName,
		},
	})
	if err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return err
	}
	service := out.Services[0]
	originalRunningTaskCount = int(*out.Services[0].RunningCount)
	// ロールアウトで置き換えられたタスクの数
	totalReplacedCnt := 0
	// ロールアウト実行回数。CreateServiceで第一陣が配置されてるので1
	totalRollOutCnt := 0
	// 推定ロールアウト施行回数
	estimatedRollOutCount := EstimateRollOutCount(originalRunningTaskCount)
	log.Infof(
		"currently %d tasks running on '%s', %d times roll out estimated",
		originalRunningTaskCount, *envars.ServiceName, estimatedRollOutCount,
	)
	lb := service.LoadBalancers[0]
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
		endTime.Add(time.Duration(*envars.RollOutPeriod) * time.Second)
		replaceCnt := EnsureReplaceCount(totalRollOutCnt, totalReplacedCnt, originalRunningTaskCount)
		// Phase1: service-nextにtask-nextを指定数配置
		log.Infof("start adding of next tasks. this will add %d tasks to %s", replaceCnt, *service.ServiceName)
		_, err := envars.StartTasks(awsEcs, envars.NextTaskDefinitionArn, replaceCnt)
		if err != nil {
			log.Errorf("failed to add next tasks due to: %s", err)
			return err
		}
		log.Infof(
			"start accumulating periodic service health of '%s' during %s ~ %s",
			*service.ServiceName, startTime.String(), endTime.String(),
		)
		// Phase2: service-nextのperiodic healthを計測
		health, err := envars.AccumulatePeriodicServiceHealth(cw, *envars.LoadBalancerArn, *lb.TargetGroupArn, startTime, endTime)
		if err != nil {
			return err
		}
		log.Infof("periodic health accumulated: availability=%f, responseTime=%f", health.availability, health.responseTime)
		if *envars.AvailabilityThreshold > health.availability || health.responseTime > *envars.ResponseTimeThreshold {
			log.Warnf(
				"😢 %dth canary test has failed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
				totalRollOutCnt, health.availability, *envars.AvailabilityThreshold, health.responseTime, *envars.ResponseTimeThreshold,
			)
			return envars.Rollback(awsEcs, originalRunningTaskCount)
		}
		log.Infof(
			"😙 %dth canary test has passed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
			totalRollOutCnt, health.availability, *envars.AvailabilityThreshold, health.responseTime, *envars.ResponseTimeThreshold,
		)
		// Phase3: service-currentからタスクを指定数消す
		log.Infof("%dth roll out starting: will replace %d tasks", totalRollOutCnt, replaceCnt)
		removed, err := envars.StopTasks(awsEcs, envars.CurrentTaskDefinitionArn, replaceCnt)
		if err != nil {
			log.Errorf("failed to roll out tasks due to: %s", err.Error())
			return err
		}
		totalReplacedCnt += len(removed)
		log.Infof(
			"%dth roll out successfully completed. %d/%d tasks roll outed",
			totalRollOutCnt, totalReplacedCnt, originalRunningTaskCount,
		)
		totalRollOutCnt += 1
		// Phase4: ロールアウトが終わったかどうかを確認
		out, err := envars.ListTasks(awsEcs, []*string{
			envars.CurrentTaskDefinitionArn,
			envars.NextTaskDefinitionArn,
		})
		if err != nil {
			log.Errorf("failed to list tasks due to: %s", err.Error())
			return err
		}
		currentTaskCount := len(out[0])
		nextTaskCount := len(out[1])
		if currentTaskCount == 0 && nextTaskCount >= originalRunningTaskCount {
			log.Infof("☀️all tasks successfully have been roll outed!☀️")
			return nil
		} else {
			log.Infof(
				"roll out is continuing. %d tasks running by '%s', %d tasks by '%s'",
				currentTaskCount, *envars.CurrentTaskDefinitionArn,
				nextTaskCount, *envars.NextTaskDefinitionArn,
			)
		}
	}
}

func (envars *Envars) StartTasks(
	awsEcs ecsiface.ECSAPI,
	taskDefinition *string,
	numToAdd int,
) ([]*ecs.Task, error) {
	eg := errgroup.Group{}
	var mux sync.Mutex
	var ret []*ecs.Task
	for i := 0; i < numToAdd; i++ {
		eg.Go(func() error {
			group := fmt.Sprintf("service:%s", *envars.ServiceName)
			out, err := awsEcs.StartTask(&ecs.StartTaskInput{
				Cluster:        envars.Cluster,
				TaskDefinition: taskDefinition,
				Group:          &group,
			})
			if err != nil {
				log.Errorf(
					"failed to start task into next service: taskArn=%s",
					*out.Tasks[0].TaskArn,
				)
				return err
			}
			if err := awsEcs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
				Cluster: envars.Cluster,
				Tasks:   []*string{out.Tasks[0].TaskArn},
			}); err != nil {
				log.Errorf(
					"launched next task state hasn't reached RUNNING state within maximum attempt windows: taskArn=%s",
					*out.Tasks[0].TaskArn,
				)
				return err
			}
			mux.Lock()
			defer mux.Unlock()
			ret = append(ret, out.Tasks...)
			log.Infof(
				"task '%s' on '%s has successfully started",
				*out.Tasks[0].TaskArn, *envars.ServiceName,
			)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Errorf("registering of next tasks hasn't completed due to: %s", err)
		return nil, err
	}
	log.Infof("registering of next tasks has successfully completed")
	return ret, nil
}

func (envars *Envars) StopTasks(
	awsEcs ecsiface.ECSAPI,
	taskDefinition *string,
	numToRemove int,
) ([]*ecs.Task, error) {
	out, err := envars.ListTasks(awsEcs, []*string{taskDefinition})
	if err != nil {
		log.Errorf("failed to list current tasks due to: %s", err.Error())
		return nil, err
	}
	log.Debugf("%s", out[0])
	tasks := out[0]
	eg := errgroup.Group{}
	var ret []*ecs.Task
	var mux sync.Mutex
	if len(tasks) < numToRemove || numToRemove < 0 {
		numToRemove = len(tasks)
	}
	for i := 0; i < numToRemove; i++ {
		task := tasks[i]
		eg.Go(func() error {
			out, err := awsEcs.StopTask(&ecs.StopTaskInput{
				Cluster: envars.Cluster,
				Task:    task.TaskArn,
			})
			if err != nil {
				log.Errorf("failed to stop task from current service: taskArn=%s", *task)
				return err
			}
			if err := awsEcs.WaitUntilTasksStopped(&ecs.DescribeTasksInput{
				Cluster: envars.Cluster,
				Tasks:   []*string{out.Task.TaskArn},
			}); err != nil {
				log.Errorf("stopped current task state hasn't reached STOPPED state within maximum attempt windows: taskArn=%s", out.Task.TaskArn)
				return err
			}
			log.Infof(
				"task '%s' on '%s' has successfully stopped",
				*task.TaskArn, *envars.ServiceName,
			)
			mux.Lock()
			defer mux.Unlock()
			ret = append(ret, out.Task)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Errorf("%dth draining of current tasks hasn't completed due to: %s", err.Error())
		return nil, err
	}
	return ret, nil
}

func (envars *Envars) ListTasks(
	awsEcs ecsiface.ECSAPI,
	taskDefinitions []*string,
) ([][]*ecs.Task, error) {
	out, err := awsEcs.ListTasks(&ecs.ListTasksInput{
		Cluster:     envars.Cluster,
		ServiceName: envars.ServiceName,
	})
	if err != nil {
		log.Errorf("failed to list current tasks due to: %s", err.Error())
		return nil, err
	}
	var ret [][]*ecs.Task
	if out, err := awsEcs.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: envars.Cluster,
		Tasks:   out.TaskArns,
	}); err != nil {
		log.Errorf("failed to describe tasks due to: %s", err)
		return nil, err
	} else {
		for _, td := range taskDefinitions {
			var res []*ecs.Task
			for _, v := range out.Tasks {
				log.Debugf("%s == %s", *td, *v.TaskDefinitionArn)
				if *v.TaskDefinitionArn == *td {
					res = append(res, v)
				}
			}
			ret = append(ret, res)
		}
		log.Debugf("%s", ret)
		return ret, nil
	}
}

func (envars *Envars) Rollback(
	awsEcs ecsiface.ECSAPI,
	originalTaskCount int,
) error {
	tasks, err := envars.ListTasks(awsEcs, []*string{envars.CurrentTaskDefinitionArn})
	if err != nil {
		return err
	}
	currentTasks := tasks[0]
	rollbackCount := originalTaskCount - len(currentTasks)
	if rollbackCount < 0 {
		rollbackCount = len(currentTasks)
	}
	log.Infof(
		"start rollback of current service: originalTaskCount=%d, currentTaskCount=%d, tasksToBeRollback=%d",
		originalTaskCount, len(currentTasks), rollbackCount,
	)
	rollbackCompletedCount := 0
	rollbackFailedCount := 0
	serviceGroup := fmt.Sprintf("service:%s", *envars.ServiceName)
	eg := errgroup.Group{}
	eg.Go(func() error {
		_, err := envars.StopTasks(awsEcs, envars.NextTaskDefinitionArn, -1)
		return err
	})
	for i := 0; i < rollbackCount; i++ {
		eg.Go(func() error {
			// タスクを追加
			out, err := awsEcs.StartTask(&ecs.StartTaskInput{
				Cluster:        envars.Cluster,
				TaskDefinition: envars.CurrentTaskDefinitionArn,
				Group:          &serviceGroup,
			})
			if err != nil {
				rollbackFailedCount += 1
				log.Errorf("failed to launch task: taskArn=%s, totalFailure=%d", *out.Tasks[0].TaskArn, rollbackFailedCount)
				return err
			}
			if err := awsEcs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
				Cluster: envars.Cluster,
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
		log.Errorf(
			"😱service rollback hasn't completed: succeeded=%d/%d, failed=%d",
			rollbackCompletedCount, rollbackCount, rollbackFailedCount,
		)
		return err
	}
	log.Info("😓service rollback has completed successfully")
	return nil
}
