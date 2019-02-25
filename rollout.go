package cage

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"time"
)

type RollOutResult struct {
	StartTime     time.Time
	EndTime       time.Time
	ServiceIntact bool
	Error         error
}

func (c *cage) RollOut(ctx context.Context) *RollOutResult {
	ret := &RollOutResult{
		StartTime:     now(),
		ServiceIntact: true,
	}
	throw := func(err error) *RollOutResult {
		ret.EndTime = now()
		ret.Error = err
		return ret
	}
	var service *ecs.Service
	if out, err := c.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: &c.cluster,
		Services: []*string{
			&c.service,
		},
	}); err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return throw(err)
	} else {
		service = out.Services[0]
	}
	if *service.LaunchType == "EC2" && c.canaryInstanceArn == nil {
		return throw(fmt.Errorf("🥺 --canaryInstanceArn is required when LaunchType = 'EC2'"))
	}
	var (
		targetGroupArn *string
	)
	if len(service.LoadBalancers) > 0 {
		targetGroupArn = service.LoadBalancers[0].TargetGroupArn
	}
	//log.Infof("ensuring next task definition...")
	//nextTaskDefinition, err := c.CreateNextTaskDefinition()
	//if err != nil {
	//	log.Errorf("failed to register next task definition due to: %s", err)
	//	return throw(err)
	//}
	log.Infof("starting canary task...")
	var canaryTask *StartCanaryTaskOutput
	if o, err := c.StartCanaryTask(c.taskDefinition); err != nil {
		log.Errorf("failed to create next service due to: %s", err)
		return throw(err)
	} else {
		canaryTask = o
	}
	log.Infof("canary task '%s' ensured.", *canaryTask.task.TaskArn)
	if targetGroupArn != nil {
		log.Infof("ensuring canary task to become healthy...")
		if err := c.EnsureTaskHealthy(
			canaryTask.task.TaskArn,
			targetGroupArn,
			canaryTask.targetId,
			canaryTask.targetPort,
		); err != nil {
			return throw(err)
		}
		log.Info("🤩 canary task is healthy!")
	}
	ret.ServiceIntact = false
	log.Infof("updating '%s' 's task definition to '%s:%d'...", c.service, *c.taskDefinition.Family, *c.taskDefinition.Revision)
	if _, err := c.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:        &c.cluster,
		Service:        &c.service,
		TaskDefinition: c.taskDefinition.TaskDefinitionArn,
	}); err != nil {
		return throw(err)
	}
	log.Infof("waiting for service '%s' to be stable...", c.service)
	if err := c.ecs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  &c.cluster,
		Services: []*string{&c.service},
	}); err != nil {
		return throw(err)
	}
	log.Infof("🥴 service '%s' has become to be stable!", c.service)
	log.Infof("deleting canary task '%s'...", *canaryTask.task.TaskArn)
	if err := c.StopCanaryTask(canaryTask); err != nil {
		return throw(err)
	}
	log.Infof("canary task '%s' has successfully been stopped", *canaryTask.task.TaskArn)
	log.Infof("🤗 service '%s' rolled out to '%s:%d'", c.service, *c.taskDefinition.Family, *c.taskDefinition.Revision)
	ret.EndTime = now()
	return ret
}

func (c *cage) EnsureTaskHealthy(
	taskArn *string,
	tgArn *string,
	targetId *string,
	targetPort *int64,
) error {
	log.Infof("checking canary task's health state...")
	var unusedCount = 0
	var initialized = false
	var recentState *string
	for {
		<-newTimer(time.Duration(15) * time.Second).C
		if o, err := c.alb.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: tgArn,
			Targets: []*elbv2.TargetDescription{{
				Id:   targetId,
				Port: targetPort,
			}},
		}); err != nil {
			return err
		} else {
			recentState = GetTargetIsHealthy(o, targetId, targetPort)
			if recentState == nil {
				return fmt.Errorf("'%s' is not registered to target group '%s'", *targetId, *tgArn)
			}
			log.Infof("canary task '%s' (%s:%d) state is: %s", *taskArn, *targetId, *targetPort, *recentState)
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
		log.Errorf("😨 canary task '%s' is unhealthy", taskArn)
		// delete canary service
		log.Infof("stopping canary task '%s'...", taskArn)
		_, err := c.ecs.StopTask(&ecs.StopTaskInput{
			Cluster: &c.cluster,
			Task:    taskArn,
			Reason:  aws.String(fmt.Sprintf("cage: canary task didn't got healthy. recent state is '%s'", *recentState)),
		})
		if err != nil {
			return fmt.Errorf("failed to stop canary task '%s': %s", *taskArn, err)
		}
		log.Infof("canary service '%s' was deleted")
	}
	return fmt.Errorf("canary task '%s' (%s:%d) hasn't become to healthy. Recent state: %s", *taskArn, *targetId, *targetPort, *recentState)
}

func GetTargetIsHealthy(o *elbv2.DescribeTargetHealthOutput, targetId *string, targetPort *int64) *string {
	for _, desc := range o.TargetHealthDescriptions {
		log.Debugf("%+v", desc)
		if *desc.Target.Id == *targetId && *desc.Target.Port == *targetPort {
			return desc.TargetHealth.State
		}
	}
	return nil
}

//func (c *cage) CreateNextTaskDefinition() (*ecs.TaskDefinition, error) {
//	if c.taskDefinition != "" {
//		log.Infof("--taskDefinitionArn was set to '%s'. skip registering new task definition.", c.env.TaskDefinitionArn)
//		o, err := c.ecs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
//			TaskDefinition: &c.env.TaskDefinitionArn,
//		})
//		if err != nil {
//			log.Errorf(
//				"failed to describe next task definition '%s' due to: %s",
//				c.env.TaskDefinitionArn, err,
//			)
//			return nil, err
//		}
//		return o.TaskDefinition, nil
//	} else {
//		if out, err := c.ecs.RegisterTaskDefinition(c.env.taskDefinition); err != nil {
//			return nil, err
//		} else {
//			return out.TaskDefinition, nil
//		}
//	}
//}

func (c *cage) DescribeSubnet(subnetId *string) (*ec2.Subnet, error) {
	if o, err := c.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: []*string{subnetId},
	}); err != nil {
		return nil, err
	} else {
		return o.Subnets[0], nil
	}
}

type StartCanaryTaskOutput struct {
	task                *ecs.Task
	registrationSkipped bool
	targetGroupArn      *string
	availabilityZone    *string
	targetId            *string
	targetPort          *int64
}

func (c *cage) StartCanaryTask(nextTaskDefinition *ecs.TaskDefinition) (*StartCanaryTaskOutput, error) {
	var service *ecs.Service
	if o, err := c.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  &c.cluster,
		Services: []*string{&c.service},
	}); err != nil {
		return nil, err
	} else {
		service = o.Services[0]
	}
	startTask := &ecs.StartTaskInput{
		Cluster:              &c.cluster,
		Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.service)),
		NetworkConfiguration: service.NetworkConfiguration,
		TaskDefinition:       nextTaskDefinition.TaskDefinitionArn,
	}
	if c.canaryInstanceArn != nil {
		// ec2
		startTask.ContainerInstances = []*string{c.canaryInstanceArn}
	}
	var task *ecs.Task
	if o, err := c.ecs.StartTask(startTask); err != nil {
		return nil, err
	} else {
		task = o.Tasks[0]
	}
	log.Infof("🥚 waiting for canary task '%s' is running...", *task.TaskArn)
	if err := c.ecs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
		Cluster: &c.cluster,
		Tasks:   []*string{task.TaskArn},
	}); err != nil {
		return nil, err
	}
	log.Infof("🐣 canary task '%s' is running!️", *task.TaskArn)
	if len(service.LoadBalancers) == 0 {
		log.Infof("no load balancers is attached to service '%s'. skip registration to target group", *service.ServiceName)
		return &StartCanaryTaskOutput{
			task:                task,
			registrationSkipped: true,
		}, nil
	}
	var targetId *string
	var targetPort *int64
	var subnet *ec2.Subnet
	for _, container := range nextTaskDefinition.ContainerDefinitions {
		if *container.Name == *service.LoadBalancers[0].ContainerName {
			targetPort = container.PortMappings[0].HostPort
		}
	}
	if *task.LaunchType == "FARGATE" {
		details := task.Attachments[0].Details
		var subnetId *string
		var privateIp *string
		for _, v := range details {
			if *v.Name == "subnetId" {
				subnetId = v.Value
			} else if *v.Name == "privateIPv4Address" {
				privateIp = v.Value
			}
		}
		if o, err := c.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
			SubnetIds: []*string{subnetId},
		}); err != nil {
			return nil, err
		} else {
			subnet = o.Subnets[0]
		}
		targetId = privateIp
		log.Infof("canary task was placed: privateIp = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	} else {
		var containerInstance *ecs.ContainerInstance
		if outputs, err := c.ecs.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            &c.cluster,
			ContainerInstances: []*string{c.canaryInstanceArn},
		}); err != nil {
			return nil, err
		} else {
			containerInstance = outputs.ContainerInstances[0]
		}
		if o, err := c.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{containerInstance.Ec2InstanceId},
		}); err != nil {
			return nil, err
		} else if sn, err := c.DescribeSubnet(o.Reservations[0].Instances[0].SubnetId); err != nil {
			return nil, err
		} else {
			targetId = containerInstance.Ec2InstanceId
			subnet = sn
		}
		log.Infof("canary task was placed: instanceId = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	}
	if _, err := c.alb.RegisterTargets(&elbv2.RegisterTargetsInput{
		TargetGroupArn: service.LoadBalancers[0].TargetGroupArn,
		Targets: []*elbv2.TargetDescription{{
			AvailabilityZone: subnet.AvailabilityZone,
			Id:               targetId,
			Port:             targetPort,
		}},
	}); err != nil {
		return nil, err
	}
	return &StartCanaryTaskOutput{
		targetId:   targetId,
		targetPort: targetPort,
		task:       task,
	}, nil
}

func (c *cage) StopCanaryTask(input *StartCanaryTaskOutput) error {
	if _, err := c.ecs.StopTask(&ecs.StopTaskInput{
		Cluster: &c.cluster,
		Task:    input.task.TaskArn,
	}); err != nil {
		return err
	}
	if _, err := c.alb.DeregisterTargets(&elbv2.DeregisterTargetsInput{
		TargetGroupArn: input.targetGroupArn,
		Targets: []*elbv2.TargetDescription{{
			AvailabilityZone: input.availabilityZone,
			Id:               input.targetId,
			Port:             input.targetPort,
		}},
	}); err != nil {
		return err
	}
	if err := c.alb.WaitUntilTargetDeregistered(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: input.targetGroupArn,
		Targets: []*elbv2.TargetDescription{{
			AvailabilityZone: input.availabilityZone,
			Id:               input.targetId,
			Port:             input.targetPort,
		}},
	}); err != nil {
		return err
	}
	if err := c.ecs.WaitUntilTasksStopped(&ecs.DescribeTasksInput{
		Cluster: &c.cluster,
		Tasks:   []*string{input.task.TaskArn},
	}); err != nil {
		return err
	}
	return nil
}
