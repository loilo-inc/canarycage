package cage

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"time"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
	"math"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"strconv"
)

type Context struct {
	Ecs ecsiface.ECSAPI
	Cw  cloudwatchiface.CloudWatchAPI
	Alb elbv2iface.ELBV2API
}

type RollOutResult struct {
	StartTime    *time.Time
	EndTime      *time.Time
	HandledError error
	Rolledback   *bool
}

func (envars *Envars) StartGradualRollOut(
	ctx *Context,
) (*RollOutResult, error) {
	ret := &RollOutResult{
		StartTime:  aws.Time(now()),
		Rolledback: aws.Bool(false),
	}
	log.Infof("ensuring next task definition...")
	nextTaskDefinition, err := envars.CreateNextTaskDefinition(ctx.Ecs)
	if err != nil {
		log.Errorf("failed to register next task definition due to: %s", err)
		return nil, err
	}
	log.Infof("ensuring next service '%s'...", *envars.NextServiceName)
	if err := envars.CreateNextService(ctx.Ecs, nextTaskDefinition.TaskDefinitionArn); err != nil {
		log.Errorf("failed to create next service due to: %s", err)
		return nil, err
	}
	// ロールバックのためのデプロイを始める前の現在のサービスのタスク数
	var originalDesiredCount int64
	out, err := ctx.Ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: envars.Cluster,
		Services: []*string{
			envars.CurrentServiceName,
			envars.NextServiceName,
		},
	})
	if err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return nil, err
	}
	currentService := out.Services[0]
	nextService := out.Services[1]
	originalDesiredCount = *currentService.DesiredCount
	var (
		targetGroupArn *string
	)
	if len(nextService.LoadBalancers) > 0 {
		targetGroupArn = nextService.LoadBalancers[0].TargetGroupArn
	}
	log.Infof("service '%s' ensured. start rolling out", *envars.NextServiceName)
	if err := envars.RollOut(ctx, targetGroupArn, originalDesiredCount); err != nil {
		log.Errorf("failed to roll out due to: %s", err)
		if err := envars.Rollback(ctx, &originalDesiredCount, targetGroupArn); err != nil {
			log.Errorf("😱 failed to rollback service '%s' due to: %s", err)
			return nil, err
		}
		ret.Rolledback = aws.Bool(true)
		ret.HandledError = err
		log.Infof("😓 service '%s' has successfully rolledback", *envars.CurrentServiceName)
	}
	ret.EndTime = aws.Time(now())
	return ret, nil
}

func (envars *Envars) CanaryTest(
	cw cloudwatchiface.CloudWatchAPI,
	loadBalancerArn *string,
	targetGroupArn *string,
	totalRollOutCnt int64,
) error {
	startTime := now()
	endTime := startTime.Add(time.Duration(*envars.RollOutPeriod) * time.Second)
	standBy := time.Duration(60-math.Floor(float64(startTime.Second()))) * time.Second
	// ロールアウトの検証期間待つ
	log.Infof(
		"waiting for %d sec and standing by %f sec for CloudWatch aggregation",
		*envars.RollOutPeriod, standBy.Seconds(),
	)
	<-newTimer(time.Duration(*envars.RollOutPeriod)*time.Second + standBy).C
	log.Infof(
		"start accumulating periodic service health of '%s' during %s ~ %s",
		*envars.NextServiceName, startTime.String(), endTime.String(),
	)
	health, err := envars.AccumulatePeriodicServiceHealth(cw, loadBalancerArn, targetGroupArn, startTime, endTime)
	if err != nil {
		return err
	}
	log.Infof("periodic health accumulated: availability=%f, responseTime=%f", health.availability, health.responseTime)
	if *envars.AvailabilityThreshold > health.availability || health.responseTime > *envars.ResponseTimeThreshold {
		return NewErrorf(
			"😢 %dth canary test has failed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
			totalRollOutCnt, health.availability, *envars.AvailabilityThreshold, health.responseTime, *envars.ResponseTimeThreshold,
		)
	}
	log.Infof(
		"😙 %dth canary test has passed: availability=%f (thresh: %f), responseTime=%f (thresh: %f)",
		totalRollOutCnt, health.availability, *envars.AvailabilityThreshold, health.responseTime, *envars.ResponseTimeThreshold,
	)
	return nil
}

func (envars *Envars) RollOut(
	ctx *Context,
	targetGroupArn *string,
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
	var (
		loadBalancerArn *string
	)
	if targetGroupArn != nil {
		o, err := ctx.Alb.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
			TargetGroupArns: []*string{targetGroupArn},
		})
		if err != nil {
			log.Errorf("failed to describe target groups due to: %s", err)
			return err
		}
		loadBalancerArn = o.TargetGroups[0].LoadBalancerArns[0]
	} else {
		// LBがないサービスはCanaryTestは行わない
		log.Infof("service '%s' has no load balancer. will skip canary tests", *envars.NextServiceName)
		envars.SkipCanary = aws.Bool(true)
	}
	// next serviceのperiodic healthが安定し、current serviceのtaskの数が0になるまで繰り返す
	for {
		log.Infof("=== preparing for %dth roll out ===", totalRollOutCnt)
		if estimatedRollOutCount <= totalRollOutCnt {
			return NewErrorf(
				"estimated roll out attempts count exceeded: estimated=%d, rolledOut=%d, replaced=%d/%d",
				estimatedRollOutCount, totalRollOutCnt, totalReplacedCnt, originalDesiredCount,
			)
		}
		replaceCnt := int64(EnsureReplaceCount(totalReplacedCnt, totalRollOutCnt, originalDesiredCount))
		scaleCnt := totalReplacedCnt + replaceCnt
		// Phase1: service-nextにtask-nextを指定数配置
		log.Infof("%dth roll out starting: will replace %d tasks", totalRollOutCnt, replaceCnt)
		log.Infof("start adding of next tasks. this will update '%s' desired count %d to %d", *envars.NextServiceName, totalReplacedCnt, scaleCnt)
		err := envars.UpdateDesiredCount(ctx, envars.NextServiceName, targetGroupArn, &originalDesiredCount, &scaleCnt, true)
		if err != nil {
			log.Errorf("failed to add next tasks due to: %s", err)
			return err
		}
		// Phase2: service-nextのperiodic healthを計測
		if *envars.SkipCanary {
			log.Infof("🤫 %dth canary test skipped.", totalRollOutCnt)
		} else if err := envars.CanaryTest(ctx.Cw, loadBalancerArn, targetGroupArn, totalRollOutCnt); err != nil {
			return err
		}
		// Phase3: service-currentからタスクを指定数消す
		descaledCnt := originalDesiredCount - totalReplacedCnt - replaceCnt
		log.Infof("updating service '%s' desired count to %d", *envars.CurrentServiceName, descaledCnt)
		if err := envars.UpdateDesiredCount(ctx, envars.CurrentServiceName, targetGroupArn, &originalDesiredCount, &descaledCnt, false); err != nil {
			log.Errorf("failed to roll out tasks due to: %s", err.Error())
			return err
		}
		totalReplacedCnt += replaceCnt
		log.Infof(
			"%dth roll out successfully completed. %d/%d tasks rolled out",
			totalRollOutCnt, totalReplacedCnt, originalDesiredCount,
		)
		totalRollOutCnt += 1
		// Phase4: ロールアウトが終わったかどうかを確認
		var (
			oldTaskCount int64
			newTaskCount int64
		)
		if out, err := ctx.Ecs.DescribeServices(&ecs.DescribeServicesInput{
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
			// ロールアウトが終わったら最終検証を行う
			log.Infof("estimated roll out completed. Do final canary test...")
			if *envars.SkipCanary {
				log.Infof("😑 final canary test skipped...")
			} else if err := envars.CanaryTest(ctx.Cw, loadBalancerArn, targetGroupArn, totalRollOutCnt); err != nil {
				log.Errorf("final canary test has failed due to: %s", err)
				return err
			}
			log.Infof("☀️all tasks successfully have been rolled out!☀️")
			// service-currentを消す
			log.Infof("deleting service '%s'...", *envars.CurrentServiceName)
			if _, err := ctx.Ecs.DeleteService(&ecs.DeleteServiceInput{
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
		service.ServiceName = envars.NextServiceName
		service.TaskDefinition = nextTaskDefinitionArn
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

func (envars *Envars) EnsureHealthyTargets(
	alb elbv2iface.ELBV2API,
	targetGroupArn *string,
	originalCount *int64,
	retryCount int,
) error {
	o, err := alb.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: targetGroupArn,
	})
	if err != nil {
		log.Errorf("failed to describe target group '%s' due to: %s", *targetGroupArn, err)
		return err
	}
	healthyCount := int64(0)
	for _, v := range o.TargetHealthDescriptions {
		if *v.TargetHealth.State == "healthy" {
			healthyCount++
		}
	}
	if healthyCount < *originalCount {
		if retryCount < 1 {
			// healthyターゲットが足りない場合は、一度だけリトライする
			log.Warnf(
				"healthy targets count in tg '%s' is less than original count(%d). retry checking after %d sec",
				*targetGroupArn, *originalCount, *envars.RollOutPeriod,
			)
			<-newTimer(time.Duration(*envars.RollOutPeriod) * time.Second).C
			return envars.EnsureHealthyTargets(alb, targetGroupArn, originalCount, retryCount+1)
		} else {
			return NewErrorf(
				"tg '%s' currently doesn't have enough healthy targets (%d/%d)",
				*targetGroupArn, healthyCount, *originalCount,
			)
		}
	}
	return nil
}

func (envars *Envars) UpdateDesiredCount(
	ctx *Context,
	serviceName *string,
	targetGroupArn *string,
	originalCount *int64,
	count *int64,
	increase bool,
) error {
	log.Infof("start ensuring healthy targets...")
	var service *ecs.Service
	if o, err := ctx.Ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      envars.Cluster,
		Service:      serviceName,
		DesiredCount: count,
	}); err != nil {
		log.Errorf("failed to update '%s' desired count to %d due to: %s", *serviceName, *count, err)
		return err
	} else {
		service = o.Service
	}
	log.Infof(
		"waiting until %d tasks running on service '%s'...",
		*count, *serviceName,
	)
	if err := ctx.Ecs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{serviceName},
	}); err != nil {
		log.Errorf("failed to wait service-stable due to: %s", err)
		return err
	}
	// LBがないサービスはここで終わり
	if targetGroupArn == nil {
		return nil
	}
	o, err := ctx.Alb.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: []*string{targetGroupArn},
	})
	if err != nil {
		log.Errorf("failed to describe target group '%s' due to: %s", *targetGroupArn, err)
		return err
	}
	tg := o.TargetGroups[0]
	var standBy time.Duration
	if increase {
		// TargetGroupに新しいタスクが登録されるまで　待つ
		standBy = time.Duration(
			*service.HealthCheckGracePeriodSeconds+*tg.HealthCheckIntervalSeconds*(*tg.HealthyThresholdCount),
		) * time.Second
		log.Infof(
			"waiting for %f seconds until new tasks are registered to target group '%s'",
			standBy.Seconds(), *tg.TargetGroupName,
		)
	} else {
		// target groupのderegistration delayだけ待つ
		var delay int64
		if o, err := ctx.Alb.DescribeTargetGroupAttributes(&elbv2.DescribeTargetGroupAttributesInput{
			TargetGroupArn: targetGroupArn,
		}); err != nil {
			return err
		} else {
			for _, v := range o.Attributes {
				if *v.Key == "deregistration_delay.timeout_seconds" {
					o, err := strconv.ParseInt(*v.Value, 10, 64)
					if err != nil {
						log.Warnf("failed to parse deregistration_delay.timeout_seconds due to: %s", err)
					} else {
						delay = o
					}
				}
			}
		}
		standBy = time.Duration(delay) * time.Second
		log.Infof(
			"waiting for %f seconds until old tasks are deregistered from target group '%s'",
			standBy.Seconds(), *tg.TargetGroupName,
		)
	}
	<-newTimer(standBy).C
	if err := envars.EnsureHealthyTargets(ctx.Alb, targetGroupArn, originalCount, 0); err != nil {
		log.Errorf("failed to ensure healthy target due to: %s", err)
		return err
	}
	return nil
}

func (envars *Envars) Rollback(
	ctx *Context,
	originalTaskCount *int64,
	targetGroupArn *string,
) error {
	// service-currentをもとのdesiredCountに戻す
	log.Infof("rollback '%s' desired count to %d", *envars.CurrentServiceName, *originalTaskCount)
	if err := envars.UpdateDesiredCount(ctx, envars.CurrentServiceName, targetGroupArn, originalTaskCount, originalTaskCount, true); err != nil {
		log.Errorf("failed to rollback desired count of %s due to: %s", *envars.CurrentServiceName, err)
		return err
	}
	// service-nextを消す
	log.Infof("'%s' desired count is now %d", *envars.NextServiceName, *originalTaskCount)
	if err := envars.UpdateDesiredCount(ctx, envars.NextServiceName, targetGroupArn, originalTaskCount, aws.Int64(0), false); err != nil {
		log.Errorf("failed to update '%s' desired count to 0 due to: %s", *envars.NextServiceName, err)
		return err
	}
	log.Infof("deleting service '%s'...", *envars.NextServiceName)
	if _, err := ctx.Ecs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: envars.Cluster,
		Service: envars.NextServiceName,
	}); err != nil {
		log.Errorf("failed to delete service '%s' due to: %s", *envars.NextServiceName, err)
		return err
	}
	log.Infof("service '%s' has successfully deleted", *envars.NextServiceName)
	log.Infof("waiting for service become to be inactive...")
	if err := ctx.Ecs.WaitUntilServicesInactive(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.NextServiceName},
	}); err != nil {
		log.Errorf("deleted service '%s' hasn't reached INACTIVE state within maximum attempt windows due to: %s", err)
		return err
	}
	log.Infof("service '%s' has been eliminated", *envars.NextServiceName)
	// TODO: 2018/08/24 ロールバック後もカナリアテストを行うべきか？ by sakurai
	return nil
}
