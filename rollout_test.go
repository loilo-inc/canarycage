package cage_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/golang/mock/gomock"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/stretchr/testify/assert"
)

func TestCage_RollOut_FARGATE(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	t.Run("basic", func(t *testing.T) {
		for _, v := range []int{1, 2, 15} {
			log.Info("====")
			envars := test.DefaultEnvars()
			ctrl := gomock.NewController(t)
			mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, v, "FARGATE")

			if mctx.ActiveServiceSize() != 1 {
				t.Fatalf("current service not setup")
			}

			if taskCnt := mctx.RunningTaskSize(); taskCnt != v {
				t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
			}

			cagecli := cage.NewCage(&cage.Input{
				Env:  envars,
				Ecs:  ecsMock,
				Alb:  albMock,
				Ec2:  ec2Mock,
				Time: test.NewFakeTime(),
			})
			ctx := context.Background()
			result, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
			assert.NoError(t, err)
			assert.False(t, result.ServiceIntact)
			assert.Equal(t, 1, mctx.ActiveServiceSize())
			assert.Equal(t, v, mctx.RunningTaskSize())
		}
	})
	t.Run("multiple load balancers", func(t *testing.T) {
		log.Info("====")
		envars := test.DefaultEnvars()
		lb := envars.ServiceDefinitionInput.LoadBalancers[0]
		envars.ServiceDefinitionInput.LoadBalancers = []ecstypes.LoadBalancer{lb, lb}
		ctrl := gomock.NewController(t)

		mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 1, "FARGATE")
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Alb:  albMock,
			Ec2:  ec2Mock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		result, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
		assert.NoError(t, err)
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, 1, mctx.ActiveServiceSize())
		assert.Equal(t, 1, mctx.RunningTaskSize())
	})
	t.Run("wait until canary task is registered to target group", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, ecsMock, _, ec2Mock := test.Setup(ctrl, envars, 2, "FARGATE")

		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTarget).Times(1)
		albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeregisterTarget).Times(1)
		albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetGroupAttibutes).Times(1)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{
						Target: &elbv2types.TargetDescription{
							Id:               aws.String("127.0.0.1"),
							Port:             aws.Int32(80),
							AvailabilityZone: aws.String("us-west-2"),
						},
						TargetHealth: &elbv2types.TargetHealth{
							State: elbv2types.TargetHealthStateEnumUnused,
						},
					}},
			}, nil).Times(2),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes(),
		)
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Alb:  albMock,
			Ec2:  ec2Mock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		result, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("stop rolloing out when canary task is not registered to target group", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, ecsMock, _, ec2Mock := test.Setup(ctrl, envars, 2, "FARGATE")
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTarget).Times(1)
		albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeregisterTarget).Times(1)
		albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetGroupAttibutes).Times(1)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{{
					Target: &elbv2types.TargetDescription{
						Id:               aws.String("192.0.0.1"),
						Port:             aws.Int32(8000),
						AvailabilityZone: aws.String("us-west-2"),
					},
					TargetHealth: &elbv2types.TargetHealth{
						State: elbv2types.TargetHealthStateEnumUnhealthy,
					},
				}, {
					Target: &elbv2types.TargetDescription{
						Id:               aws.String("127.0.0.1"),
						Port:             aws.Int32(8000),
						AvailabilityZone: aws.String("us-west-2"),
					},
					TargetHealth: &elbv2types.TargetHealth{
						State: elbv2types.TargetHealthStateEnumUnused,
					},
				}},
			}, nil),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes(),
		)
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Ec2:  ec2Mock,
			Alb:  albMock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		_, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
		assert.NotNil(t, err)
	})
	t.Run("update service", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 1, "FARGATE")
		newLb := ecstypes.LoadBalancer{
			ContainerName:  aws.String("container"),
			ContainerPort:  aws.Int32(80),
			TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/new-target-group/abcdefg"),
		}
		newNetwork := &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        []string{"subnet-1234567890abcdefg"},
				SecurityGroups: []string{"sg-12345678"},
			},
		}
		envars.ServiceDefinitionInput.LoadBalancers = []ecstypes.LoadBalancer{newLb}
		envars.ServiceDefinitionInput.NetworkConfiguration = newNetwork
		envars.ServiceDefinitionInput.PlatformVersion = aws.String("LATEST")
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Alb:  albMock,
			Ec2:  ec2Mock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		service, _ := mctx.GetService(envars.Service)
		assert.Equal(t, "1.4.0", *service.PlatformVersion)
		assert.NotNil(t, service.NetworkConfiguration)
		assert.NotNil(t, service.LoadBalancers)
		_, err := cagecli.RollOut(ctx, &cage.RollOutInput{UpdateService: true})
		assert.NoError(t, err)
		service, _ = mctx.GetService(envars.Service)
		assert.Equal(t, "LATEST", *service.PlatformVersion)
		assert.Equal(t, *newNetwork, *service.NetworkConfiguration)
		assert.Equal(t, *service.LoadBalancers[0].ContainerName, *newLb.ContainerName)
	})
	t.Run("should stop canary task if error occurs before registering target", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 1, "FARGATE")
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Alb:  albMock,
			Ec2:  ec2Mock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		envars.ServiceDefinitionInput.LoadBalancers = []ecstypes.LoadBalancer{
			{
				ContainerName:  aws.String("missing-container"),
				ContainerPort:  aws.Int32(80),
				TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/new-target-group/abcdefg"),
			},
		}
		result, err := cagecli.RollOut(ctx, &cage.RollOutInput{UpdateService: true})
		assert.EqualError(t, err, "failed to wait for canary task due to: couldn't find host port in container definition")
		assert.Equal(t, result.ServiceIntact, true)
		assert.Equal(t, 1, mctx.RunningTaskSize())
	})
	t.Run("Show error if service doesn't exist", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 2, "FARGATE")
		delete(mocker.Services, envars.Service)
		cagecli := cage.NewCage(&cage.Input{
			Env: envars,
			Ecs: ecsMock,
			Ec2: ec2Mock,
			Alb: albMock,
		})
		ctx := context.Background()
		_, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
		assert.EqualError(t, err, "service 'service' doesn't exist. Run 'cage up' or create service before rolling out")
	})
	t.Run("Roll out even if the service does not have a load balancer", func(t *testing.T) {
		envars := test.DefaultEnvars()
		envars.ServiceDefinitionInput.LoadBalancers = nil
		envars.CanaryTaskIdleDuration = 1
		ctrl := gomock.NewController(t)
		_, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 2, "FARGATE")
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Alb:  albMock,
			Ec2:  ec2Mock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		if res, err := cagecli.RollOut(ctx, &cage.RollOutInput{}); err != nil {
			t.Fatalf(err.Error())
		} else if res.ServiceIntact {
			t.Fatalf("no")
		}
	})

	t.Run("stop rolloing out when service status is inactive", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{Status: aws.String("INACTIVE")},
				},
			}, nil,
		)
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Time: test.NewFakeTime(),
		})
		_, err := cagecli.RollOut(context.Background(), &cage.RollOutInput{})
		assert.EqualError(t, err, "😵 'service' status is 'INACTIVE'. Stop rolling out")
	})
	t.Run("Stop rolling out if the canary task container does not become healthy", func(t *testing.T) {
		envars := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, _, albMock, ec2Mock := test.Setup(ctrl, envars, 2, "FARGATE")

		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().CreateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.CreateService).AnyTimes()
		ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.UpdateService).AnyTimes()
		ecsMock.EXPECT().DeleteService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeleteService).AnyTimes()
		ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.StartTask).AnyTimes()
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTaskDefinition).AnyTimes()
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeServices).AnyTimes()
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, input *ecs.DescribeTasksInput, opts ...func(options *ecs.Options)) (*ecs.DescribeTasksOutput, error) {
				out, err := mocker.DescribeTasks(ctx, input, opts...)
				if err != nil {
					return out, err
				}

				task := mocker.Tasks[input.Tasks[0]]
				if strings.Contains(*task.Group, "canary-task") {
					for i := range out.Tasks {
						for i2 := range out.Tasks[i].Containers {
							out.Tasks[i].Containers[i2].HealthStatus = ecstypes.HealthStatusUnknown
						}
					}
				}
				return out, err
			},
		).AnyTimes()
		ecsMock.EXPECT().ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.ListTasks).AnyTimes()
		ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeContainerInstances).AnyTimes()
		ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RunTask).AnyTimes()
		ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.StopTask).AnyTimes()

		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Ec2:  ec2Mock,
			Alb:  albMock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		res, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
		assert.NotNil(t, res)
		assert.NotNil(t, err)

		for _, task := range mocker.Tasks {
			if strings.Contains(*task.Group, "canary-task") {
				assert.Equal(t, "STOPPED", *task.LastStatus)
			}
		}
	})

}

func TestCage_RollOut_EC2(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	for _, v := range []int{1, 2, 15} {
		log.Info("====")
		canaryInstanceArn := "arn:aws:ecs:us-west-2:1234567689012:container-instance/abcdefg-hijk-lmn-opqrstuvwxyz"
		attributeValue := "true"
		envars := test.DefaultEnvars()
		envars.CanaryInstanceArn = canaryInstanceArn
		ctrl := gomock.NewController(t)
		mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, v, "ec2")
		ecsMock.EXPECT().ListAttributes(gomock.Any(), gomock.Any()).Return(&ecs.ListAttributesOutput{
			Attributes: []ecstypes.Attribute{
				{
					Name:     &envars.Service,
					Value:    &attributeValue,
					TargetId: &canaryInstanceArn,
				},
			},
		}, nil).AnyTimes()
		if mctx.ActiveServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := mctx.RunningTaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		cagecli := cage.NewCage(&cage.Input{
			Env:  envars,
			Ecs:  ecsMock,
			Ec2:  ec2Mock,
			Alb:  albMock,
			Time: test.NewFakeTime(),
		})
		ctx := context.Background()
		result, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
		if err != nil {
			t.Fatalf("%s", err)
		}
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, 1, mctx.ActiveServiceSize())
		assert.Equal(t, v, mctx.RunningTaskSize())
	}
}

func TestCage_RollOut_EC2_without_ContainerInstanceArn(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	log.Info("====")
	envars := test.DefaultEnvars()
	ctrl := gomock.NewController(t)
	mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 1, "EC2")
	if mctx.ActiveServiceSize() != 1 {
		t.Fatalf("current service not setup")
	}
	if taskCnt := mctx.RunningTaskSize(); taskCnt != 1 {
		t.Fatalf("current tasks not setup: %d/%d", 1, taskCnt)
	}
	cagecli := cage.NewCage(&cage.Input{
		Env:  envars,
		Ecs:  ecsMock,
		Ec2:  ec2Mock,
		Alb:  albMock,
		Time: test.NewFakeTime(),
	})
	ctx := context.Background()
	result, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
	if err == nil {
		t.Fatal("Rollout with no container instance should be error")
	} else {
		assert.True(t, regexp.MustCompile("canaryInstanceArn is required").MatchString(err.Error()))
		assert.NotNil(t, result)
	}
}

func TestCage_RollOut_EC2_no_attribute(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	log.Info("====")
	canaryInstanceArn := "arn:aws:ecs:us-west-2:1234567689012:container-instance/abcdefg-hijk-lmn-opqrstuvwxyz"
	envars := test.DefaultEnvars()
	envars.CanaryInstanceArn = canaryInstanceArn
	ctrl := gomock.NewController(t)
	mctx, ecsMock, albMock, ec2Mock := test.Setup(ctrl, envars, 1, "EC2")
	if mctx.ActiveServiceSize() != 1 {
		t.Fatalf("current service not setup")
	}
	if taskCnt := mctx.RunningTaskSize(); taskCnt != 1 {
		t.Fatalf("current tasks not setup: %d/%d", 1, taskCnt)
	}
	ecsMock.EXPECT().ListAttributes(gomock.Any(), gomock.Any()).Return(&ecs.ListAttributesOutput{
		Attributes: []ecstypes.Attribute{},
	}, nil).AnyTimes()
	ecsMock.EXPECT().PutAttributes(gomock.Any(), gomock.Any()).Return(&ecs.PutAttributesOutput{}, nil).AnyTimes()
	cagecli := cage.NewCage(&cage.Input{
		Env:  envars,
		Ecs:  ecsMock,
		Ec2:  ec2Mock,
		Alb:  albMock,
		Time: test.NewFakeTime(),
	})
	ctx := context.Background()
	result, err := cagecli.RollOut(ctx, &cage.RollOutInput{})
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.False(t, result.ServiceIntact)
	assert.Equal(t, 1, mctx.ActiveServiceSize())
	assert.Equal(t, 1, mctx.RunningTaskSize())
}
