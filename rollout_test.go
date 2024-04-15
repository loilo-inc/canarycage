package cage

import (
	"context"
	"encoding/json"
	"io/ioutil"
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
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/stretchr/testify/assert"
)

func DefaultEnvars() *Envars {
	d, _ := ioutil.ReadFile("fixtures/task-definition.json")
	var taskDefinition ecs.RegisterTaskDefinitionInput
	if err := json.Unmarshal(d, &taskDefinition); err != nil {
		log.Fatalf(err.Error())
	}
	return &Envars{
		Region:                 "us-west-2",
		Cluster:                "cage-test",
		Service:                "service",
		ServiceDefinitionInput: ReadServiceDefinition("fixtures/service.json"),
		TaskDefinitionInput:    &taskDefinition,
	}
}

func ReadServiceDefinition(path string) *ecs.CreateServiceInput {
	d, _ := ioutil.ReadFile(path)
	var dest ecs.CreateServiceInput
	if err := json.Unmarshal(d, &dest); err != nil {
		log.Fatalf(err.Error())
	}
	return &dest
}

func Setup(ctrl *gomock.Controller, envars *Envars, currentTaskCount int, launchType string) (
	*test.MockContext,
	*mock_awsiface.MockEcsClient,
	*mock_awsiface.MockAlbClient,
	*mock_awsiface.MockEc2Client,
) {
	mocker := test.NewMockContext()

	ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
	ecsMock.EXPECT().CreateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.CreateService).AnyTimes()
	ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.UpdateService).AnyTimes()
	ecsMock.EXPECT().DeleteService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeleteService).AnyTimes()
	ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.StartTask).AnyTimes()
	ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTaskDefinition).AnyTimes()
	ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeServices).AnyTimes()
	ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTasks).AnyTimes()
	ecsMock.EXPECT().ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.ListTasks).AnyTimes()
	ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeContainerInstances).AnyTimes()
	ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RunTask).AnyTimes()
	ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.StopTask).AnyTimes()

	albMock := mock_awsiface.NewMockAlbClient(ctrl)
	albMock.EXPECT().DescribeTargetGroups(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetGroups).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes()
	albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetGroupAttibutes).AnyTimes()
	albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
	albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()

	ec2Mock := mock_awsiface.NewMockEc2Client(ctrl)
	ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeSubnets).AnyTimes()
	ec2Mock.EXPECT().DescribeInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeInstances).AnyTimes()
	td, _ := mocker.RegisterTaskDefinition(context.Background(), envars.TaskDefinitionInput)
	a := &ecs.CreateServiceInput{
		ServiceName:    &envars.Service,
		LoadBalancers:  envars.ServiceDefinitionInput.LoadBalancers,
		TaskDefinition: td.TaskDefinition.TaskDefinitionArn,
		DesiredCount:   aws.Int32(int32(currentTaskCount)),
		LaunchType:     ecstypes.LaunchType(launchType),
	}
	svc, _ := mocker.CreateService(context.Background(), a)
	if len(svc.Service.LoadBalancers) > 0 {
		_, _ = mocker.RegisterTarget(context.Background(), &elbv2.RegisterTargetsInput{
			TargetGroupArn: svc.Service.LoadBalancers[0].TargetGroupArn,
		})
	}
	return mocker, ecsMock, albMock, ec2Mock
}

func TestCage_RollOut_FARGATE(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	t.Cleanup(recoverTimer)

	t.Run("basic", func(t *testing.T) {
		for _, v := range []int{1, 2, 15} {
			log.Info("====")
			envars := DefaultEnvars()
			ctrl := gomock.NewController(t)
			mctx, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, v, "FARGATE")

			if mctx.ServiceSize() != 1 {
				t.Fatalf("current service not setup")
			}

			if taskCnt := mctx.RunningTaskSize(); taskCnt != v {
				t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
			}

			cagecli := NewCage(&Input{
				Env: envars,
				ECS: ecsMock,
				ALB: albMock,
				EC2: ec2Mock,
			})
			ctx := context.Background()
			result, err := cagecli.RollOut(ctx)
			if err != nil {
				t.Fatalf("%s", err)
			}
			assert.False(t, result.ServiceIntact)
			assert.Equal(t, 1, mctx.ServiceSize())
			assert.Equal(t, v, mctx.RunningTaskSize())
		}
	})

	t.Run("canary taskがtgに登録されるまで少し待つ", func(t *testing.T) {
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, ecsMock, _, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")

		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
		albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()
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
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			ALB: albMock,
			EC2: ec2Mock,
		})
		ctx := context.Background()
		result, err := cagecli.RollOut(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}
		assert.NotNil(t, result)
	})

	t.Run("canary taskがtgに登録されない場合は打ち切る", func(t *testing.T) {
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, ecsMock, _, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
		albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()
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
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			EC2: ec2Mock,
			ALB: albMock,
		})
		ctx := context.Background()
		_, err := cagecli.RollOut(ctx)
		assert.NotNil(t, err)
	})

	t.Run("Show error if service doesn't exist", func(t *testing.T) {
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")
		delete(mocker.Services, envars.Service)
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			EC2: ec2Mock,
			ALB: albMock,
		})
		ctx := context.Background()
		_, err := cagecli.RollOut(ctx)
		assert.EqualError(t, err, "service 'service' doesn't exist. Run 'cage up' or create service before rolling out")
	})

	t.Run("lbがないサービスの場合もロールアウトする", func(t *testing.T) {
		envars := DefaultEnvars()
		envars.ServiceDefinitionInput.LoadBalancers = nil
		envars.CanaryTaskIdleDuration = 1
		ctrl := gomock.NewController(t)
		_, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			EC2: ec2Mock,
			ALB: albMock,
		})
		ctx := context.Background()
		if res, err := cagecli.RollOut(ctx); err != nil {
			t.Fatalf(err.Error())
		} else if res.ServiceIntact {
			t.Fatalf("no")
		}
	})

	t.Run("stop rolloing out when service status is inactive", func(t *testing.T) {
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{Status: aws.String("INACTIVE")},
				},
			}, nil,
		)
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
		})
		_, err := cagecli.RollOut(context.Background())
		assert.EqualError(t, err, "😵 'service' status is 'INACTIVE'. Stop rolling out")
	})

	t.Run("canary task container が healthy にならない場合は打ち切る", func(t *testing.T) {
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mocker, _, albMock, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")

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

		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			EC2: ec2Mock,
			ALB: albMock,
		})
		ctx := context.Background()
		res, err := cagecli.RollOut(ctx)
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
	newTimer = fakeTimer
	defer recoverTimer()
	for _, v := range []int{1, 2, 15} {
		log.Info("====")
		canaryInstanceArn := "arn:aws:ecs:us-west-2:1234567689012:container-instance/abcdefg-hijk-lmn-opqrstuvwxyz"
		attributeValue := "true"
		envars := DefaultEnvars()
		envars.CanaryInstanceArn = canaryInstanceArn
		ctrl := gomock.NewController(t)
		mctx, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, v, "ec2")
		ecsMock.EXPECT().ListAttributes(gomock.Any(), gomock.Any()).Return(&ecs.ListAttributesOutput{
			Attributes: []ecstypes.Attribute{
				{
					Name:     &envars.Service,
					Value:    &attributeValue,
					TargetId: &canaryInstanceArn,
				},
			},
		}, nil).AnyTimes()
		if mctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := mctx.RunningTaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			EC2: ec2Mock,
			ALB: albMock,
		})
		ctx := context.Background()
		result, err := cagecli.RollOut(ctx)
		if err != nil {
			t.Fatalf("%s", err)
		}
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, 1, mctx.ServiceSize())
		assert.Equal(t, v, mctx.RunningTaskSize())
	}
}

func TestCage_RollOut_EC2_without_ContainerInstanceArn(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	log.Info("====")
	envars := DefaultEnvars()
	ctrl := gomock.NewController(t)
	mctx, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, 1, "EC2")
	if mctx.ServiceSize() != 1 {
		t.Fatalf("current service not setup")
	}
	if taskCnt := mctx.RunningTaskSize(); taskCnt != 1 {
		t.Fatalf("current tasks not setup: %d/%d", 1, taskCnt)
	}
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		EC2: ec2Mock,
		ALB: albMock,
	})
	ctx := context.Background()
	result, err := cagecli.RollOut(ctx)
	if err == nil {
		t.Fatal("Rollout with no container instance should be error")
	} else {
		assert.True(t, regexp.MustCompile("canaryInstanceArn is required").MatchString(err.Error()))
		assert.NotNil(t, result)
	}
}

func TestCage_RollOut_EC2_no_attribute(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	log.Info("====")
	canaryInstanceArn := "arn:aws:ecs:us-west-2:1234567689012:container-instance/abcdefg-hijk-lmn-opqrstuvwxyz"
	envars := DefaultEnvars()
	envars.CanaryInstanceArn = canaryInstanceArn
	ctrl := gomock.NewController(t)
	mctx, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, 1, "EC2")
	if mctx.ServiceSize() != 1 {
		t.Fatalf("current service not setup")
	}
	if taskCnt := mctx.RunningTaskSize(); taskCnt != 1 {
		t.Fatalf("current tasks not setup: %d/%d", 1, taskCnt)
	}
	ecsMock.EXPECT().ListAttributes(gomock.Any(), gomock.Any()).Return(&ecs.ListAttributesOutput{
		Attributes: []ecstypes.Attribute{},
	}, nil).AnyTimes()
	ecsMock.EXPECT().PutAttributes(gomock.Any(), gomock.Any()).Return(&ecs.PutAttributesOutput{}, nil).AnyTimes()
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		EC2: ec2Mock,
		ALB: albMock,
	})
	ctx := context.Background()
	result, err := cagecli.RollOut(ctx)
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.False(t, result.ServiceIntact)
	assert.Equal(t, 1, mctx.ServiceSize())
	assert.Equal(t, 1, mctx.RunningTaskSize())
}

func TestCage_CreateNextTaskDefinition(t *testing.T) {
	envars := &Envars{
		TaskDefinitionArn: "arn://task",
	}
	ctrl := gomock.NewController(t)
	e := mock_awsiface.NewMockEcsClient(ctrl)
	e.EXPECT().DescribeTaskDefinition(gomock.Any(), gomock.Any()).Return(
		&ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: &ecstypes.TaskDefinition{TaskDefinitionArn: aws.String("arn://task")},
		}, nil)
	// nextTaskDefinitionArnがある場合はdescribeTaskDefinitionから返す
	cagecli := &cage{env: envars, ecs: e}
	o, err := cagecli.CreateNextTaskDefinition(context.Background())
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, envars.TaskDefinitionArn, *o.TaskDefinitionArn)
}
