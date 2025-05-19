package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewAlbTask(t *testing.T) {
	d := &di.D{}
	input := &Input{}
	lb := &ecstypes.LoadBalancer{}
	task := NewAlbTask(d, input, lb)
	v, ok := task.(*albTask)
	assert.NotNil(t, task)
	assert.True(t, ok)
	assert.Equal(t, input, v.Input)
	assert.Equal(t, lb, v.lb)
}

func TestAlbTask(t *testing.T) {
	setup := func(env *env.Envars) (*albTask, *test.MockContext) {
		mocker := test.NewMockContext()
		ctx := context.TODO()
		td, _ := mocker.Ecs.RegisterTaskDefinition(ctx, env.TaskDefinitionInput)
		env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
		ecsSvc, _ := mocker.Ecs.CreateService(ctx, env.ServiceDefinitionInput)
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, mocker.Ecs)
			b.Set(key.Ec2Cli, mocker.Ec2)
			b.Set(key.AlbCli, mocker.Alb)
			b.Set(key.Time, test.NewFakeTime())
		})
		stask := &albTask{
			lb: &ecsSvc.Service.LoadBalancers[0],
			common: &common{
				di: d,
				Input: &Input{
					TaskDefinition:       td.TaskDefinition,
					NetworkConfiguration: ecsSvc.Service.NetworkConfiguration,
				},
			},
		}
		mocker.Alb.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
			TargetGroupArn: ecsSvc.Service.LoadBalancers[0].TargetGroupArn,
		})
		return stask, mocker
	}
	t.Run("fargate", func(t *testing.T) {
		env := test.DefaultEnvars()
		stask, mocker := setup(env)
		ctx := context.TODO()
		err := stask.Start(ctx)
		assert.NoError(t, err)
		err = stask.Wait(ctx)
		assert.NoError(t, err)
		err = stask.Stop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, mocker.RunningTaskSize())
	})
	t.Run("ec2", func(t *testing.T) {
		env := test.DefaultEnvars()
		env.CanaryInstanceArn = "arn://ec2"
		stask, mocker := setup(env)
		ctx := context.TODO()
		err := stask.Start(ctx)
		assert.NoError(t, err)
		err = stask.Wait(ctx)
		assert.NoError(t, err)
		err = stask.Stop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, mocker.RunningTaskSize())
	})
}

func TestAlbTask_Wait(t *testing.T) {
	t.Run("should error if task is not running", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		cm := &albTask{
			common: &common{
				taskArn: aws.String("task-arn"),
				Input:   &Input{},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, test.DefaultEnvars())
					b.Set(key.EcsCli, ecsMock)
				}),
			},
		}
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("STOPPED")}},
			}, nil)
		err := cm.Wait(context.TODO())
		assert.ErrorContains(t, err, "failed to wait for canary task to be running")
	})
	t.Run("should error if container is not healthy", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		mocker := test.NewMockContext()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), test.DefaultEnvars().TaskDefinitionInput)
		env := test.DefaultEnvars()
		env.CanaryTaskHealthCheckWait = 1
		cm := &albTask{
			common: &common{
				taskArn: aws.String("task-arn"),
				Input:   &Input{TaskDefinition: td.TaskDefinition},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, env)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Time, test.NewFakeTime())
				}),
			},
		}
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{{
						Name:         env.TaskDefinitionInput.ContainerDefinitions[0].Name,
						HealthStatus: ecstypes.HealthStatusUnhealthy,
					}},
				}},
			}, nil).Times(2)
		err := cm.Wait(context.TODO())
		assert.ErrorContains(t, err, "canary task hasn't become to be healthy")
	})
	t.Run("should erro if RegisterToTargetGroup failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		mocker := test.NewMockContext()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), test.DefaultEnvars().TaskDefinitionInput)
		env := test.DefaultEnvars()
		env.CanaryTaskHealthCheckWait = 1
		cm := &albTask{
			common: &common{
				taskArn: aws.String("task-arn"),
				Input: &Input{
					TaskDefinition:       td.TaskDefinition,
					NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, env)
					b.Set(key.EcsCli, mocker.Ecs)
					b.Set(key.AlbCli, albMock)
					b.Set(key.Ec2Cli, mocker.Ec2)
					b.Set(key.Time, test.NewFakeTime())
				}),
			},
			lb: &env.ServiceDefinitionInput.LoadBalancers[0],
		}
		albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
		err := cm.Start(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
		err = cm.Wait(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
	})
	t.Run("should error if waitUntilTargetHealthy failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		mocker := test.NewMockContext()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), test.DefaultEnvars().TaskDefinitionInput)
		env := test.DefaultEnvars()
		env.CanaryTaskHealthCheckWait = 1
		cm := &albTask{
			common: &common{
				taskArn: aws.String("task-arn"),
				Input: &Input{
					TaskDefinition:       td.TaskDefinition,
					NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, env)
					b.Set(key.EcsCli, mocker.Ecs)
					b.Set(key.AlbCli, albMock)
					b.Set(key.Ec2Cli, mocker.Ec2)
					b.Set(key.Time, test.NewFakeTime())
				}),
			},
			lb: &env.ServiceDefinitionInput.LoadBalancers[0],
		}
		albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any()).Return(nil, nil)
		albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
		err := cm.Start(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
		err = cm.Wait(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
	})
}

func TestAlbTask_WaitUntilTargetHealthy(t *testing.T) {
	target := &elbv2types.TargetDescription{
		Id:               aws.String("127.0.0.1"),
		Port:             aws.Int32(80),
		AvailabilityZone: aws.String("ap-northeast-1a"),
	}
	setup := func(t *testing.T) (*mock_awsiface.MockAlbClient, *albTask) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		mocker := test.NewMockContext()
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), env.TaskDefinitionInput)
		atask := &albTask{
			common: &common{
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.AlbCli, albMock)
					b.Set(key.Time, test.NewFakeTime())
				}),
				Input: &Input{
					TaskDefinition:       td.TaskDefinition,
					NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration,
				}},
			lb: &env.ServiceDefinitionInput.LoadBalancers[0],
		}
		atask.taskArn = aws.String("arn://task")
		atask.target = target
		return albMock, atask
	}
	t.Run("should not count as unhealthy if target is initial", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumInitial},
						Target: target,
					},
				},
			}, nil).Times(5),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumHealthy},
						Target: target,
					},
				},
			}, nil).Times(1),
		)
		err := atask.waitUntilTargetHealthy(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should call DescribeTargetHealth periodically", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnused},
						Target: target,
					},
				},
			}, nil).Times(1),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumHealthy},
						Target: target,
					},
				},
			}, nil).Times(1),
		)
		err := atask.waitUntilTargetHealthy(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if DescribeTargetHealth failed", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1),
		)
		err := atask.waitUntilTargetHealthy(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
	})
	t.Run("should error if context is canceled", func(t *testing.T) {
		_, atask := setup(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := atask.waitUntilTargetHealthy(ctx)
		assert.EqualError(t, err, "context canceled")
	})
	t.Run("should error if target is not registered", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{},
			}, nil).Times(1),
		)
		err := atask.waitUntilTargetHealthy(context.TODO())
		assert.EqualError(t, err, fmt.Sprintf(
			"'%s' is not registered to the target group '%s'", *target.Id, *atask.lb.TargetGroupArn),
		)
	})
	t.Run("should error if target unhelthy counts exceed the limit", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnhealthy},
						Target: target,
					},
				},
			}, nil).Times(5),
		)
		err := atask.waitUntilTargetHealthy(context.TODO())
		assert.EqualError(t, err, fmt.Sprintf(
			"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
			*atask.taskArn, *target.Id, *target.Port, elbv2types.TargetHealthStateEnumUnhealthy,
		),
		)
	})
}

func TestAlbTask_RegisterToTargetGroup(t *testing.T) {
	t.Run("should error if port mapping is not found", func(t *testing.T) {
		env := test.DefaultEnvars()
		mocker := test.NewMockContext()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), env.TaskDefinitionInput)
		atask := &albTask{
			common: &common{
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, env)
				}),
				Input: &Input{
					TaskDefinition:       td.TaskDefinition,
					NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration,
				},
			},
			lb: &ecstypes.LoadBalancer{
				TargetGroupArn: aws.String("arn://target-group"),
				ContainerName:  aws.String("unknown")},
		}
		atask.taskArn = aws.String("arn://task")
		err := atask.registerToTargetGroup(context.TODO())
		assert.EqualError(t, err, "couldn't find host port in container definition")
	})
	t.Run("Fargate", func(t *testing.T) {
		attachments := []ecstypes.Attachment{{
			Status: aws.String("ATTACHED"),
			Type:   aws.String("ElasticNetworkInterface"),
			Details: []ecstypes.KeyValuePair{{
				Name:  aws.String("networkInterfaceId"),
				Value: aws.String("eni-123456"),
			}, {
				Name:  aws.String("subnetId"),
				Value: aws.String("subnet-123456"),
			}, {
				Name:  aws.String("privateIPv4Address"),
				Value: aws.String("127.0.0.1"),
			},
			}}}
		subnets := []ec2types.Subnet{{
			AvailabilityZone: aws.String("ap-northeast-1a"),
		}}
		setup := func(t *testing.T) (*mock_awsiface.MockEc2Client, *mock_awsiface.MockAlbClient, *mock_awsiface.MockEcsClient, *albTask) {
			ctrl := gomock.NewController(t)
			envars := test.DefaultEnvars()
			mocker := test.NewMockContext()
			td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
			ec2Mock := mock_awsiface.NewMockEc2Client(ctrl)
			albMock := mock_awsiface.NewMockAlbClient(ctrl)
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			atask := &albTask{
				common: &common{
					di: di.NewDomain(func(b *di.B) {
						b.Set(key.Env, envars)
						b.Set(key.Ec2Cli, ec2Mock)
						b.Set(key.AlbCli, albMock)
						b.Set(key.EcsCli, ecsMock)
					}),
					Input: &Input{
						TaskDefinition:       td.TaskDefinition,
						NetworkConfiguration: envars.ServiceDefinitionInput.NetworkConfiguration,
					},
				},
				lb: &envars.ServiceDefinitionInput.LoadBalancers[0],
			}
			atask.taskArn = aws.String("arn://task")
			return ec2Mock, albMock, ecsMock, atask
		}
		t.Run("should call RegisterTargets", func(t *testing.T) {
			ec2Mock, albMock, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{
					LastStatus:  aws.String("RUNNING"),
					Attachments: attachments},
				}}, nil)
			ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(&ec2.DescribeSubnetsOutput{
				Subnets: subnets,
			}, nil)
			albMock.EXPECT().RegisterTargets(gomock.Any(), &elbv2.RegisterTargetsInput{
				TargetGroupArn: atask.lb.TargetGroupArn,
				Targets: []elbv2types.TargetDescription{{
					Id:               aws.String("127.0.0.1"),
					Port:             aws.Int32(80),
					AvailabilityZone: subnets[0].AvailabilityZone},
				}}).Return(nil, nil)
			atask.taskArn = aws.String("arn://task")
			err := atask.registerToTargetGroup(context.TODO())
			assert.NoError(t, err)
		})
		t.Run("should error if DescribeTasks failed", func(t *testing.T) {
			_, _, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, assert.AnError.Error())
		})
		t.Run("should error if DescribeSubnets failed", func(t *testing.T) {
			ec2Mock, _, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{
					LastStatus:  aws.String("RUNNING"),
					Attachments: attachments},
				}}, nil)
			ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, assert.AnError.Error())
		})
		t.Run("should error if task is not attached to the network interface", func(t *testing.T) {
			_, _, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{
					LastStatus: aws.String("RUNNING"),
				}},
			}, nil)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, "couldn't find ElasticNetworkInterface attachment in task")
		})
		t.Run("should error if RegisterTargets failed", func(t *testing.T) {
			ec2Mock, albMock, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{
					LastStatus:  aws.String("RUNNING"),
					Attachments: attachments},
				}}, nil)
			ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(&ec2.DescribeSubnetsOutput{
				Subnets: subnets,
			}, nil)
			albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, assert.AnError.Error())
		})
	})
	t.Run("EC2", func(t *testing.T) {
		containerInstances := []ecstypes.ContainerInstance{{
			ContainerInstanceArn: aws.String("arn://container"),
			Ec2InstanceId:        aws.String("i-123456"),
		}}
		reservations := []ec2types.Reservation{{
			Instances: []ec2types.Instance{{
				InstanceId:       aws.String("i-123456"),
				SubnetId:         aws.String("subnet-123456"),
				PrivateIpAddress: aws.String("127.0.0.1"),
			}},
		}}
		subnets := []ec2types.Subnet{{
			AvailabilityZone: aws.String("ap-northeast-1a"),
		}}
		setup := func(t *testing.T) (*mock_awsiface.MockEc2Client, *mock_awsiface.MockAlbClient, *mock_awsiface.MockEcsClient, *albTask) {
			ctrl := gomock.NewController(t)
			envars := test.DefaultEnvars()
			envars.CanaryInstanceArn = "arn://container"
			mocker := test.NewMockContext()
			td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
			ec2Mock := mock_awsiface.NewMockEc2Client(ctrl)
			albMock := mock_awsiface.NewMockAlbClient(ctrl)
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			atask := &albTask{
				common: &common{
					di: di.NewDomain(func(b *di.B) {
						b.Set(key.Env, envars)
						b.Set(key.Ec2Cli, ec2Mock)
						b.Set(key.AlbCli, albMock)
						b.Set(key.EcsCli, ecsMock)
					}),
					Input: &Input{
						TaskDefinition:       td.TaskDefinition,
						NetworkConfiguration: envars.ServiceDefinitionInput.NetworkConfiguration,
					},
				},
				lb: &envars.ServiceDefinitionInput.LoadBalancers[0],
			}
			atask.taskArn = aws.String("arn://task")
			return ec2Mock, albMock, ecsMock, atask
		}
		t.Run("should call RegisterTargets", func(t *testing.T) {
			ec2Mock, albMock, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any()).Return(&ecs.DescribeContainerInstancesOutput{
				ContainerInstances: containerInstances,
			}, nil)
			ec2Mock.EXPECT().DescribeInstances(gomock.Any(), gomock.Any()).Return(&ec2.DescribeInstancesOutput{
				Reservations: reservations,
			}, nil)
			ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(&ec2.DescribeSubnetsOutput{
				Subnets: subnets,
			}, nil)
			albMock.EXPECT().RegisterTargets(gomock.Any(), &elbv2.RegisterTargetsInput{
				TargetGroupArn: atask.lb.TargetGroupArn,
				Targets: []elbv2types.TargetDescription{{
					Id:               containerInstances[0].Ec2InstanceId,
					Port:             aws.Int32(80),
					AvailabilityZone: subnets[0].AvailabilityZone},
				}}).Return(nil, nil)
			err := atask.registerToTargetGroup(context.TODO())
			assert.NoError(t, err)
		})
		t.Run("should error if DescribeContainerInstances failed", func(t *testing.T) {
			_, _, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, assert.AnError.Error())
		})
		t.Run("should error if DescribeInstances failed", func(t *testing.T) {
			ec2Mock, _, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any()).Return(&ecs.DescribeContainerInstancesOutput{
				ContainerInstances: containerInstances,
			}, nil)
			ec2Mock.EXPECT().DescribeInstances(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, assert.AnError.Error())
		})
		t.Run("should error if DescribeSubnets failed", func(t *testing.T) {
			ec2Mock, _, ecsMock, atask := setup(t)
			ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any()).Return(&ecs.DescribeContainerInstancesOutput{
				ContainerInstances: containerInstances,
			}, nil)
			ec2Mock.EXPECT().DescribeInstances(gomock.Any(), gomock.Any()).Return(&ec2.DescribeInstancesOutput{
				Reservations: reservations,
			}, nil)
			ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			err := atask.registerToTargetGroup(context.TODO())
			assert.EqualError(t, err, assert.AnError.Error())
		})
	})
}

func TestAlbTask_GetTargetDeregistrationDelay(t *testing.T) {
	setup := func(t *testing.T) (*mock_awsiface.MockAlbClient, *albTask) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		atask := &albTask{
			common: &common{
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.AlbCli, albMock)
				}),
				Input: &Input{},
			},
			lb: &env.ServiceDefinitionInput.LoadBalancers[0],
		}
		return albMock, atask
	}
	t.Run("should return deregistration delay", func(t *testing.T) {
		albMock, atask := setup(t)
		albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetGroupAttributesOutput{
			Attributes: []elbv2types.TargetGroupAttribute{
				{Key: aws.String("deregistration_delay.timeout_seconds"), Value: aws.String("100")},
			},
		}, nil)
		delay, err := atask.getTargetDeregistrationDelay(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, 100*time.Second, delay)
	})
	t.Run("should return default delay if deregistration_delay is not found", func(t *testing.T) {
		albMock, atask := setup(t)
		albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetGroupAttributesOutput{
			Attributes: []elbv2types.TargetGroupAttribute{},
		}, nil)
		delay, err := atask.getTargetDeregistrationDelay(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, 300*time.Second, delay)
	})
	t.Run("should return default delay if deregistration_delay is not a number", func(t *testing.T) {
		albMock, atask := setup(t)
		albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetGroupAttributesOutput{
			Attributes: []elbv2types.TargetGroupAttribute{
				{Key: aws.String("deregistration_delay.timeout_seconds"), Value: aws.String("invalid")},
			},
		}, nil)
		delay, err := atask.getTargetDeregistrationDelay(context.TODO())
		assert.Error(t, err)
		assert.Equal(t, 300*time.Second, delay)
	})
	t.Run("should error if DescribeTargetGroupAttributes failed", func(t *testing.T) {
		albMock, atask := setup(t)
		albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
		delay, err := atask.getTargetDeregistrationDelay(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
		assert.Equal(t, 300*time.Second, delay)
	})
}

func TestAlbTask_DeregisterTarget(t *testing.T) {
	target := &elbv2types.TargetDescription{
		Id:               aws.String("127.0.0.1"),
		Port:             aws.Int32(80),
		AvailabilityZone: aws.String("ap-northeast-1a"),
	}
	setup := func(t *testing.T, env *env.Envars) (*mock_awsiface.MockAlbClient, *albTask) {
		ctrl := gomock.NewController(t)
		mocker := test.NewMockContext()
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), env.TaskDefinitionInput)
		atask := &albTask{
			common: &common{
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.AlbCli, albMock)
				}),
				Input: &Input{
					TaskDefinition:       td.TaskDefinition,
					NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration,
				},
			},
			lb: &env.ServiceDefinitionInput.LoadBalancers[0],
		}
		atask.taskArn = aws.String("arn://task")
		atask.target = target
		return albMock, atask
	}
	t.Run("should do nothing if target is nil", func(t *testing.T) {
		atask := &albTask{}
		atask.deregisterTarget(context.TODO())
	})
	t.Run("should call DeregisterTargets and wait", func(t *testing.T) {
		env := test.DefaultEnvars()
		albMock, atask := setup(t, env)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetGroupAttributesOutput{
				Attributes: []elbv2types.TargetGroupAttribute{
					{Key: aws.String("deregistration_delay.timeout_seconds"), Value: aws.String("300")},
				},
			}, nil).Times(1),
			albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnused},
						Target: target,
					},
				},
			}, nil).Times(1),
		)
		atask.deregisterTarget(context.TODO())
	})
	t.Run("should call DeregisterTargets even if getTargetDeregistrationDelay failed", func(t *testing.T) {
		env := test.DefaultEnvars()
		albMock, atask := setup(t, env)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1),
			albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnused},
						Target: target,
					},
				},
			}, nil).Times(1),
		)
		atask.deregisterTarget(context.TODO())
	})
	t.Run("should return even if DeregisterTargets failed", func(t *testing.T) {
		env := test.DefaultEnvars()
		albMock, atask := setup(t, env)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetGroupAttributesOutput{
				Attributes: []elbv2types.TargetGroupAttribute{
					{Key: aws.String("deregistration_delay.timeout_seconds"), Value: aws.String("300")},
				},
			}, nil).Times(1),
			albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1),
		)
		atask.deregisterTarget(context.TODO())
	})
	t.Run("should return even if deregistration wait counts exceed the limit", func(t *testing.T) {
		env := test.DefaultEnvars()
		albMock, atask := setup(t, env)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetGroupAttributesOutput{
				Attributes: []elbv2types.TargetGroupAttribute{
					{Key: aws.String("deregistration_delay.timeout_seconds"), Value: aws.String("1")},
				},
			}, nil).Times(1),
			albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1),
		)
		atask.deregisterTarget(context.TODO())
	})
}
