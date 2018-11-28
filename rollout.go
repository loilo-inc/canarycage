package cage

import (
	"encoding/base64"
	"encoding/json"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"time"
)

type Context struct {
	Ecs ecsiface.ECSAPI
	Alb elbv2iface.ELBV2API
}

type RollOutResult struct {
	StartTime     time.Time
	EndTime       time.Time
	ServiceIntact bool
	Error         error
}

func (envars *Envars) RollOut(
	ctx *Context,
) (*RollOutResult) {
	ret := &RollOutResult{
		StartTime:     now(),
		ServiceIntact: true,
	}
	throw := func(err error) (*RollOutResult) {
		ret.EndTime = now()
		ret.Error = err
		return ret
	}
	out, err := ctx.Ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: envars.Cluster,
		Services: []*string{
			envars.Service,
		},
	})
	if err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return throw(err)
	}
	service := out.Services[0]
	var (
		targetGroupArn *string
	)
	if len(service.LoadBalancers) > 0 {
		targetGroupArn = service.LoadBalancers[0].TargetGroupArn
	}
	log.Infof("ensuring next task definition...")
	nextTaskDefinition, err := envars.CreateNextTaskDefinition(ctx.Ecs)
	if err != nil {
		log.Errorf("failed to register next task definition due to: %s", err)
		return throw(err)
	}
	log.Infof("ensuring canary service '%s'...", *envars.CanaryService)
	if err := envars.CreateCanaryService(ctx.Ecs, nextTaskDefinition.TaskDefinitionArn); err != nil {
		log.Errorf("failed to create next service due to: %s", err)
		return throw(err)
	}
	log.Infof("service '%s' ensured.", *envars.CanaryService)
	if targetGroupArn != nil {
		log.Infof("ensuring canary task to become healthy...")
		if err := envars.EnsureTaskHealthy(ctx, targetGroupArn); err != nil {
			return throw(err)
		}
		log.Info("🤩 canary task is healthy!")
	}
	ret.ServiceIntact = false
	log.Infof("updating '%s' 's task definition to '%s:%d'...", *envars.Service, *nextTaskDefinition.Family, *nextTaskDefinition.Revision)
	if _, err := ctx.Ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:        envars.Cluster,
		Service:        envars.Service,
		TaskDefinition: nextTaskDefinition.TaskDefinitionArn,
	}); err != nil {
		return throw(err)
	}
	log.Infof("waiting for service '%s' to be stable...", *envars.Service)
	if err := ctx.Ecs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.Service},
	}); err != nil {
		return throw(err)
	}
	log.Infof("🥴 service '%s' has become to be stable!", *envars.Service)
	log.Infof("deleting canary service '%s'...", *envars.CanaryService)
	if _, err := ctx.Ecs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: envars.Cluster,
		Service: envars.CanaryService,
		Force: aws.Bool(true),
	}); err != nil {
		return throw(err)
	}
	log.Infof("canary service '%s' has successfully deleted", *envars.CanaryService)
	log.Infof("🤗 service '%s' rolled out to '%s:%d'", *envars.Service, *nextTaskDefinition.Family, *nextTaskDefinition.Revision)
	ret.EndTime = now()
	return ret
}

func (envars *Envars) EnsureTaskHealthy(
	ctx *Context,
	tgArn *string,
) error {
	var canaryTaskPrivateIP *string
	var canaryTaskArn *string
	if o, err := ctx.Ecs.ListTasks(&ecs.ListTasksInput{
		Cluster:     envars.Cluster,
		ServiceName: envars.CanaryService,
	}); err != nil {
		return err
	} else if o, err := ctx.Ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: envars.Cluster,
		Tasks:   o.TaskArns,
	}); err != nil {
		return err
	} else {
		canaryTaskArn = o.Tasks[0].TaskArn
		for _, d := range o.Tasks[0].Attachments[0].Details {
			if *d.Name == "privateIPv4Address" {
				canaryTaskPrivateIP = d.Value
				break
			}
		}
	}
	log.Infof("checking canary task's health state...")
	var unusedCount = 0
	var initialized = false
	var recentState *string
	for {
		<-newTimer(time.Duration(15)*  time.Second).C
		if o, err := ctx.Alb.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: tgArn,
			Targets: []*elbv2.TargetDescription{{
				Id: canaryTaskPrivateIP,
			}},
		}); err != nil {
			return err
		} else {
			recentState = GetTargetIsHealthy(o, canaryTaskPrivateIP)
			if recentState == nil {
				return NewErrorf("'%s' is not registered to target group '%s'", *canaryTaskPrivateIP, *tgArn)
			}
			log.Infof("canary task '%s' (%s) state is: %s", *canaryTaskArn, *canaryTaskPrivateIP, *recentState)
			switch *recentState {
			case "healthy":
				return nil
			case "initial":
				initialized = true
				log.Infof("still checking state...")
				continue
			case "unused":
				// 20回以上=300秒間unusedになった場合はエラーにする
				unusedCount++
				if !initialized && unusedCount < 20 {
					continue
				}
			default:
			}
		}
		// unhealthy, draining, unused
		return NewErrorf("canary task '%s' (%s) hasn't become to healthy. Recent state: %s", *canaryTaskArn, *canaryTaskPrivateIP, *recentState)
	}
}

func GetTargetIsHealthy(o *elbv2.DescribeTargetHealthOutput, targetIp *string) *string {
	for _, desc := range o.TargetHealthDescriptions {
		if *desc.Target.Id == *targetIp {
			return desc.TargetHealth.State
		}
	}
	return nil
}

func (envars *Envars) CreateNextTaskDefinition(awsEcs ecsiface.ECSAPI) (*ecs.TaskDefinition, error) {
	if !isEmpty(envars.TaskDefinitionArn) {
		o, err := awsEcs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: envars.TaskDefinitionArn,
		})
		if err != nil {
			log.Errorf(
				"failed to describe next task definition '%s' due to: %s",
				*envars.TaskDefinitionArn, err,
			)
			return nil, err
		}
		return o.TaskDefinition, nil
	}
	data, err := base64.StdEncoding.DecodeString(*envars.TaskDefinitionBase64)
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

func (envars *Envars) CreateCanaryService(
	awsEcs ecsiface.ECSAPI,
	nextTaskDefinitionArn *string,
) (error) {
	service := &ecs.CreateServiceInput{}
	if envars.ServiceDefinitionBase64 == nil {
		// サービス定義が与えられなかった場合はタスク定義と名前だけ変えたservice-currentのレプリカを作成する
		log.Infof("nextServiceDefinitionBase64 not provided. try to create replica service")
		out, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  envars.Cluster,
			Services: []*string{envars.Service},
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
			ServiceName:                   envars.CanaryService,
			ServiceRegistries:             s.ServiceRegistries,
			TaskDefinition:                nextTaskDefinitionArn,
		}
	} else {
		data, err := base64.StdEncoding.DecodeString(*envars.ServiceDefinitionBase64)
		if err != nil {
			log.Errorf("failed to decode service definition base64 due to : %s", err)
			return err
		}
		if err := json.Unmarshal(data, service); err != nil {
			log.Errorf("failed to unmarshal service definition base64 due to: %s", err)
			return err
		}
		service.ServiceName = envars.CanaryService
		service.TaskDefinition = nextTaskDefinitionArn
		*service.DesiredCount = 1
	}
	log.Infof("creating canary service '%s' with desiredCount=1", *envars.CanaryService)
	if _, err := awsEcs.CreateService(service); err != nil {
		log.Errorf("failed to create canary service due to: %s", err)
		return err
	}
	log.Infof("standing up for 10 seconds for '%s' become to be ready...", *service.ServiceName)
	<-newTimer(time.Duration(10) * time.Second).C
	log.Infof("waiting for service '%s' to become STABLE", *envars.CanaryService)
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CanaryService},
	}); err != nil {
		log.Errorf("'%s' hasn't reached STABLE state within maximum attempt windows due to: %s", err)
		return err
	}
	log.Infof("service '%s' has reached STABLE state", *envars.CanaryService)
	return nil
}
