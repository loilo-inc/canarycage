package cage

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
	"errors"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
)

type ServiceHealth struct {
	availability float64
	responseTime float64
}

type NoDataPointsFoundError struct {
	Input *cloudwatch.GetMetricStatisticsInput
}

func (v *NoDataPointsFoundError) Error() string {
	return ""
}

func (envars *Envars) GetServiceMetricStatistics(
	cw cloudwatchiface.CloudWatchAPI,
	lbId string,
	tgId string,
	metricName string,
	unit string,
	startTime time.Time,
	endTime time.Time,
) (float64, error) {
	log.Infof(
		"getStatistics: LoadBalancer=%s, TargetGroup=%s, metricName=%s, unit=%s",
		lbId, tgId, metricName, unit,
	)
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace: aws.String("AWS/ApplicationELB"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("LoadBalancer"),
				Value: aws.String(lbId),
			}, {
				Name:  aws.String("TargetGroup"),
				Value: aws.String(tgId),
			},
		},
		Statistics: []*string{&unit},
		MetricName: &metricName,
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     envars.RollOutPeriod,
	}
	out, err := cw.GetMetricStatistics(input)
	if err != nil {
		log.Errorf("failed to get CloudWatch's '%s' metric statistics due to: %s", metricName, err.Error())
		return 0, err
	}
	if len(out.Datapoints) == 0 {
		return 0, &NoDataPointsFoundError{Input: input}
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
		ret /= float64(len(out.Datapoints))
	default:
		err = errors.New(fmt.Sprintf("unsuported unit type: %s", unit))
	}
	return ret, err
}
func (envars *Envars) AccumulatePeriodicServiceHealth(
	cw cloudwatchiface.CloudWatchAPI,
	targetGroupArn string,
	startTime time.Time,
	endTime time.Time,
) (*ServiceHealth, error) {
	// ロールアウトの検証期間待つ
	log.Infof("waiting for roll out period (%d sec)", *envars.RollOutPeriod)
	<-time.NewTimer(time.Duration(*envars.RollOutPeriod) * time.Second).C
	log.Infof("start accumulating periodic service health...")
	var (
		lbId string
		tgId string
		err  error
	)
	if lbId, err = ExtractAlbId(*envars.LoadBalancerArn); err != nil {
		return nil, err
	}
	if tgId, err = ExtractTargetGroupId(targetGroupArn); err != nil {
		return nil, err
	}
	timeout := time.Now().Add(time.Duration(*envars.RollOutPeriod) * time.Second)
	for time.Now().Before(timeout) {
		eg := errgroup.Group{}
		requestCnt := 0.0
		elb5xxCnt := 0.0
		target5xxCnt := 0.0
		responseTime := 0.0
		accumulate := func(metricName string, unit string, dest *float64) func() (error) {
			return func() (error) {
				if v, err := envars.GetServiceMetricStatistics(cw, lbId, tgId, metricName, unit, startTime, endTime); err != nil {
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
		err := eg.Wait()
		if err != nil {
			switch err.(type) {
			case *NoDataPointsFoundError:
				// タイミングによってCloudWatchのメトリクスデータポイントがまだ存在しない場合がある
				log.Warnf(
					"no data points found on CloudWatch Metrics between %s ~ %s. will retry after %d seconds",
					startTime.String(), endTime.String(), 15,
				)
				<-time.NewTimer(time.Duration(15) * time.Second).C
				continue
			default:
				log.Errorf("failed to accumulate periodic service health due to: %s", err.Error())
				return nil, err
			}
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
	return nil, errors.New("unknown error occurred while accumulating periodic service health")
}

func (envars *Envars) StartGradualRollOut(awsEcs ecsiface.ECSAPI, cw cloudwatchiface.CloudWatchAPI) (error) {
	log.Infof("ensuring next task definition...")
	nextTaskDefinition, err := envars.CreateNextTaskDefinition(awsEcs)
	if err != nil {
		log.Errorf("failed to register next task definition due to: %s", err)
		return err
	}
	log.Infof("ensuring next service '%s'...", *envars.NextServiceName)
	if err := envars.CreateNextService(awsEcs, nextTaskDefinition.TaskDefinitionArn); err != nil {
		log.Errorf("failed to create next service due to: %s", err)
		return err
	}
	// ロールバックのためのデプロイを始める前の現在のサービスのタスク数
	var originalDesiredCount int64
	out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: envars.Cluster,
		Services: []*string{
			envars.CurrentServiceName,
		},
	})
	if err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return err
	}
	service := out.Services[0]
	originalDesiredCount = *service.DesiredCount
	log.Infof("service '%s' ensured. start rolling out", *envars.NextServiceName)
	if err := envars.RollOut(awsEcs, cw, service, originalDesiredCount); err != nil {
		log.Errorf("failed to roll out due to: %s", err)
		err := envars.Rollback(awsEcs, &originalDesiredCount)
		if err != nil {
			log.Errorf("😱 failed to rollback service '%s' due to: %s", err)
			return err
		}
		log.Infof("😓 service '%s' has successfully rolledback", *envars.CurrentServiceName)
	}
	return nil
}

func (envars *Envars) RollOut(
	awsEcs ecsiface.ECSAPI,
	cw cloudwatchiface.CloudWatchAPI,
	nextService *ecs.Service,
	originalDesiredCount int64,
) (error) {
	var (
		// ロールアウトで置き換えられたタスクの数
		totalReplacedCnt int64 = 0
		// 総ロールアウト実行回数
		totalRollOutCnt int64 = 0
		// 推定ロールアウト施行回数
		estimatedRollOutCount = EstimateRollOutCount(originalDesiredCount)
	)
	log.Infof(
		"currently %d tasks running on '%s', %d times roll out estimated",
		originalDesiredCount, *envars.CurrentServiceName, estimatedRollOutCount,
	)
	lb := nextService.LoadBalancers[0]
	// next serviceのperiodic healthが安定し、current serviceのtaskの数が0になるまで繰り返す
	for {
		log.Infof("=== preparing for %dth roll out ===", totalRollOutCnt)
		if estimatedRollOutCount <= totalRollOutCnt {
			return errors.New(
				fmt.Sprintf(
					"estimated roll out attempts count exceeded: estimated=%d, rollouted=%d, replaced=%d/%d",
					estimatedRollOutCount, totalRollOutCnt, totalReplacedCnt, originalDesiredCount,
				),
			)
		}
		replaceCnt := int64(EnsureReplaceCount(totalReplacedCnt, totalRollOutCnt, originalDesiredCount))
		scaleCnt := totalReplacedCnt + replaceCnt
		// Phase1: service-nextにtask-nextを指定数配置
		log.Infof("%dth roll out starting: will replace %d tasks", totalRollOutCnt, replaceCnt)
		log.Infof("start adding of next tasks. this will update '%s' desired count %d to %d", *nextService.ServiceName, totalReplacedCnt, scaleCnt)
		err := envars.UpdateDesiredCount(awsEcs, envars.NextServiceName, &scaleCnt, true)
		if err != nil {
			log.Errorf("failed to add next tasks due to: %s", err)
			return err
		}
		// Phase2: service-nextのperiodic healthを計測
		startTime := time.Now()
		endTime := startTime.Add(time.Duration(*envars.RollOutPeriod) * time.Second)
		log.Infof(
			"start accumulating periodic service health of '%s' during %s ~ %s",
			*nextService.ServiceName, startTime.String(), endTime.String(),
		)
		health, err := envars.AccumulatePeriodicServiceHealth(cw, *lb.TargetGroupArn, startTime, endTime)
		if health == nil {
			return err
		}
		log.Infof("periodic health accumulated: availability=%f, responseTime=%f", health.availability, health.responseTime)
		if *envars.AvailabilityThreshold > health.availability || health.responseTime > *envars.ResponseTimeThreshold {
			return errors.New(fmt.Sprintf(
				"😢 %dth canary test has failed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
				totalRollOutCnt, health.availability, *envars.AvailabilityThreshold, health.responseTime, *envars.ResponseTimeThreshold,
			))
		}
		log.Infof(
			"😙 %dth canary test has passed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
			totalRollOutCnt, health.availability, *envars.AvailabilityThreshold, health.responseTime, *envars.ResponseTimeThreshold,
		)
		// Phase3: service-currentからタスクを指定数消す
		descaledCnt := originalDesiredCount - totalReplacedCnt - replaceCnt
		log.Infof("updating service '%s' desired count to %d", *envars.CurrentServiceName, descaledCnt)
		if err := envars.UpdateDesiredCount(awsEcs, envars.CurrentServiceName, &descaledCnt, false); err != nil {
			log.Errorf("failed to roll out tasks due to: %s", err.Error())
			return err
		}
		totalReplacedCnt += replaceCnt
		log.Infof(
			"%dth roll out successfully completed. %d/%d tasks roll outed",
			totalRollOutCnt, totalReplacedCnt, originalDesiredCount,
		)
		totalRollOutCnt += 1
		// Phase4: ロールアウトが終わったかどうかを確認
		var (
			oldTaskCount int64
			newTaskCount int64
		)
		if out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster: envars.Cluster,
			Services: []*string{
				envars.CurrentServiceName,
				envars.NextServiceName,
			},
		}); err != nil {
			log.Errorf("failed to list tasks due to: %s", err.Error())
			return err
		} else {
			oldTaskCount = *out.Services[0].DesiredCount
			newTaskCount = *out.Services[1].DesiredCount
		}
		if oldTaskCount == 0 && newTaskCount >= originalDesiredCount {
			log.Infof("☀️all tasks successfully have been roll outed!☀️")
			// service-currentを消す
			log.Infof("deleting service '%s'...", *envars.CurrentServiceName)
			if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
				Cluster: envars.Cluster,
				Service: envars.CurrentServiceName,
			}); err != nil {
				log.Errorf("failed to delete service '%s due to: %s'", *envars.CurrentServiceName, err)
				return err
			}
			log.Infof("service '%s' has been successfully deleted", *envars.CurrentServiceName)
			return nil
		} else {
			log.Infof(
				"roll out is continuing. %d tasks running on '%s', %d tasks on '%s'",
				oldTaskCount, *envars.CurrentServiceName,
				newTaskCount, *envars.NextServiceName,
			)
		}
	}
}

func (envars *Envars) CreateNextTaskDefinition(awsEcs ecsiface.ECSAPI) (*ecs.TaskDefinition, error) {
	if !isEmpty(envars.NextTaskDefinitionArn) {
		o, err := awsEcs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: envars.NextTaskDefinitionArn,
		})
		if err != nil {
			log.Errorf(
				"failed to describe next task definition '%s' due to: %s",
				*envars.NextTaskDefinitionArn, err,
			)
			return nil, err
		}
		return o.TaskDefinition, nil
	}
	data, err := base64.StdEncoding.DecodeString(*envars.NextTaskDefinitionBase64)
	if err != nil {
		log.Errorf("failed to decode task definition base64 due to :%s", err)
		return nil, err
	}
	td := &ecs.RegisterTaskDefinitionInput{}
	if err := json.Unmarshal(data, td); err != nil {
		log.Errorf("failed to unmarshal task definition due to: %s", err)
		return nil, err
	}
	if out, err := awsEcs.RegisterTaskDefinition(td); err != nil {
		return nil, err
	} else {
		return out.TaskDefinition, nil
	}
}

func (envars *Envars) CreateNextService(awsEcs ecsiface.ECSAPI, nextTaskDefinitionArn *string) (error) {
	service := &ecs.CreateServiceInput{}
	if envars.NextServiceDefinitionBase64 == nil {
		// サービス定義が与えられなかった場合はタスク定義と名前だけ変えたservice-currentのレプリカを作成する
		log.Infof("nextServiceDefinitionBase64 not provided. try to create replica service")
		out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  envars.Cluster,
			Services: []*string{envars.CurrentServiceName},
		})
		if len(out.Failures) > 0 || err != nil {
			log.Errorf("failed to describe current service due to: %s", err)
			return err
		}
		s := out.Services[0]
		service = &ecs.CreateServiceInput{
			Cluster:                       envars.Cluster,
			DeploymentConfiguration:       s.DeploymentConfiguration,
			DesiredCount:                  aws.Int64(1),
			HealthCheckGracePeriodSeconds: s.HealthCheckGracePeriodSeconds,
			LaunchType:                    s.LaunchType,
			LoadBalancers:                 s.LoadBalancers,
			NetworkConfiguration:          s.NetworkConfiguration,
			PlacementConstraints:          s.PlacementConstraints,
			PlacementStrategy:             s.PlacementStrategy,
			PlatformVersion:               s.PlatformVersion,
			SchedulingStrategy:            s.SchedulingStrategy,
			ServiceName:                   envars.NextServiceName,
			ServiceRegistries:             s.ServiceRegistries,
			TaskDefinition:                nextTaskDefinitionArn,
		}
	} else {
		data, err := base64.StdEncoding.DecodeString(*envars.NextServiceDefinitionBase64)
		if err != nil {
			log.Errorf("failed to decode service definition base64 due to : %s", err)
			return err
		}
		if err := json.Unmarshal(data, service); err != nil {
			log.Errorf("failed to unmarshal service definition base64 due to: %s", err)
			return err
		}
		*service.DesiredCount = 1
	}
	log.Infof("creating next service '%s' with desiredCount=1", *envars.NextServiceName)
	if _, err := awsEcs.CreateService(service); err != nil {
		log.Errorf("failed to create next service due to: %s", err)
		return err
	}
	log.Infof("waiting for service '%s' to become STABLE", *envars.NextServiceName)
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.NextServiceName},
	}); err != nil {
		log.Errorf("'%s' hasn't reached STABLE state within maximum attempt windows due to: %s", err)
		return err
	}
	log.Infof("service '%s' has reached STABLE state", *envars.NextServiceName)
	return nil
}

func (envars *Envars) UpdateDesiredCount(
	awsEcs ecsiface.ECSAPI,
	serviceName *string,
	count *int64,
	increase bool,
) error {
	if _, err := awsEcs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      envars.Cluster,
		Service:      serviceName,
		DesiredCount: count,
	}); err != nil {
		log.Errorf("failed to update '%s' desired count to %d due to: %s", *serviceName, *count, err)
		return err
	}
	log.Infof(
		"waiting until %d tasks running on service '%s'...",
		*count, *serviceName,
	)
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{serviceName},
	}); err != nil {
		log.Errorf("failed to wait service-stable due to: %s", err)
		return err
	}
	return nil
}

func (envars *Envars) Rollback(
	awsEcs ecsiface.ECSAPI,
	originalTaskCount *int64,
) error {
	// service-currentをもとのdesiredCountに戻す
	log.Infof("rollback '%s' desired count to %d", *envars.CurrentServiceName, *originalTaskCount)
	if err := envars.UpdateDesiredCount(awsEcs, envars.CurrentServiceName, originalTaskCount, true); err != nil {
		log.Errorf("failed to rollback desired count of %s due to: %s", *envars.CurrentServiceName, err)
		return err
	}
	// service-nextを消す
	log.Infof("'%s' desired count is now %d", *envars.NextServiceName, *originalTaskCount)
	if err := envars.UpdateDesiredCount(awsEcs, envars.NextServiceName, aws.Int64(0), false); err != nil {
		log.Errorf("failed to update '%s' desired count to 0 due to: %s", *envars.NextServiceName, err)
		return err
	}
	log.Infof("deleting service '%s'...", *envars.NextServiceName)
	if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: envars.Cluster,
		Service: envars.NextServiceName,
	}); err != nil {
		log.Errorf("failed to delete service '%s' due to: %s", *envars.NextServiceName, err)
		return err
	}
	log.Infof("service '%s' has successfully deleted", *envars.NextServiceName)
	log.Infof("waiting for service become to be inactive...")
	if err := awsEcs.WaitUntilServicesInactive(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.NextServiceName},
	}); err != nil {
		log.Errorf("deleted service '%s' hasn't reached INACTIVE state within maximum attempt windows due to: %s", err)
		return err
	}
	log.Infof("service '%s' has been eliminated", *envars.NextServiceName)
	return nil
}
