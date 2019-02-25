package cage

import (
	"context"
	"encoding/json"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mocks/github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/loilo-inc/canarycage/mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/loilo-inc/canarycage/mocks/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"regexp"
	"testing"
)

func DefaultEnvars() *Envars {
	d, _ := ioutil.ReadFile("fixtures/task-definition.json")
	var taskDefinition ecs.RegisterTaskDefinitionInput
	if err := json.Unmarshal(d, &taskDefinition); err != nil {
		log.Fatalf(err.Error())
	}
	return &Envars{
		Region:            "us-west-2",
		Cluster:           "cage-test",
		Service:           "service",
		taskDefinition:    &taskDefinition,
		serviceDefinition: ReadServiceDefinition("fixtures/service.json"),
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

func Setup(ctrl *gomock.Controller, envars *Envars, currentTaskCount int64, launchType string) (
	*test.MockContext,
	*mock_ecsiface.MockECSAPI,
	*mock_elbv2iface.MockELBV2API,
	*mock_ec2iface.MockEC2API,
) {
	ecsMock := mock_ecsiface.NewMockECSAPI(ctrl)
	albMock := mock_elbv2iface.NewMockELBV2API(ctrl)
	ec2Mock := mock_ec2iface.NewMockEC2API(ctrl)
	mocker := test.NewMockContext()
	ecsMock.EXPECT().CreateService(gomock.Any()).DoAndReturn(mocker.CreateService).AnyTimes()
	ecsMock.EXPECT().UpdateService(gomock.Any()).DoAndReturn(mocker.UpdateService).AnyTimes()
	ecsMock.EXPECT().DeleteService(gomock.Any()).DoAndReturn(mocker.DeleteService).AnyTimes()
	ecsMock.EXPECT().StartTask(gomock.Any()).DoAndReturn(mocker.StartTask).AnyTimes()
	ecsMock.EXPECT().StopTask(gomock.Any()).DoAndReturn(mocker.StopTask).AnyTimes()
	ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any()).DoAndReturn(mocker.RegisterTaskDefinition).AnyTimes()
	ecsMock.EXPECT().WaitUntilServicesStable(gomock.Any()).DoAndReturn(mocker.WaitUntilServicesStable).AnyTimes()
	ecsMock.EXPECT().WaitUntilServicesInactive(gomock.Any()).DoAndReturn(mocker.WaitUntilServicesInactive).AnyTimes()
	ecsMock.EXPECT().DescribeServices(gomock.Any()).DoAndReturn(mocker.DescribeServices).AnyTimes()
	ecsMock.EXPECT().DescribeTasks(gomock.Any()).DoAndReturn(mocker.DescribeTasks).AnyTimes()
	ecsMock.EXPECT().WaitUntilTasksRunning(gomock.Any()).DoAndReturn(mocker.WaitUntilTasksRunning).AnyTimes()
	ecsMock.EXPECT().WaitUntilTasksStopped(gomock.Any()).DoAndReturn(mocker.WaitUntilTasksStopped).AnyTimes()
	ecsMock.EXPECT().ListTasks(gomock.Any()).DoAndReturn(mocker.ListTasks).AnyTimes()
	ecsMock.EXPECT().DescribeContainerInstances(gomock.Any()).DoAndReturn(mocker.DescribeContainerInstances).AnyTimes()
	albMock.EXPECT().DescribeTargetGroups(gomock.Any()).DoAndReturn(mocker.DescribeTargetGroups).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes()
	albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any()).DoAndReturn(mocker.DescribeTargetGroupAttibutes).AnyTimes()
	albMock.EXPECT().RegisterTargets(gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
	albMock.EXPECT().DeregisterTargets(gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()
	albMock.EXPECT().WaitUntilTargetDeregistered(gomock.Any()).Return(nil).AnyTimes()
	ec2Mock.EXPECT().DescribeSubnets(gomock.Any()).DoAndReturn(mocker.DescribeSubnets).AnyTimes()
	ec2Mock.EXPECT().DescribeInstances(gomock.Any()).DoAndReturn(mocker.DescribeInstances).AnyTimes()

	td, _ := mocker.RegisterTaskDefinition(envars.taskDefinition)
	a := &ecs.CreateServiceInput{
		ServiceName:    &envars.Service,
		LoadBalancers:  envars.serviceDefinition.LoadBalancers,
		TaskDefinition: td.TaskDefinition.TaskDefinitionArn,
		DesiredCount:   aws.Int64(currentTaskCount),
		LaunchType:     aws.String(launchType),
	}
	_, _ = mocker.CreateService(a)
	return mocker, ecsMock, albMock, ec2Mock
}

func TestCage_RollOut(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	for _, v := range []int64{1, 2, 15} {
		log.Info("====")
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mctx, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, v, "FARGATE")
		if mctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := mctx.TaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			ALB: albMock,
			EC2: ec2Mock,
		})
		ctx := context.Background()
		result := cagecli.RollOut(ctx)
		if result.Error != nil {
			t.Fatalf("%s", result.Error)
		}
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, int64(1), mctx.ServiceSize())
		assert.Equal(t, v, mctx.TaskSize())
	}
}

func TestCage_RollOut2(t *testing.T) {
	// canary taskがtgに登録されるまで少し待つ
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	envars.serviceDefinition = ReadServiceDefinition("fixtures/service.json")
	ctrl := gomock.NewController(t)
	mocker, ecsMock, _, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")
	albMock := mock_elbv2iface.NewMockELBV2API(ctrl)
	albMock.EXPECT().RegisterTargets(gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
	albMock.EXPECT().DeregisterTargets(gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()
	albMock.EXPECT().WaitUntilTargetDeregistered(gomock.Any()).Return(nil).AnyTimes()
	gomock.InOrder(
		albMock.EXPECT().DescribeTargetHealth(gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
			TargetHealthDescriptions: []*elbv2.TargetHealthDescription{{
				Target: &elbv2.TargetDescription{
					Id:               aws.String("127.0.0.1"),
					Port:             aws.Int64(80),
					AvailabilityZone: aws.String("us-west-2"),
				},
				TargetHealth: &elbv2.TargetHealth{
					State: aws.String("unused"),
				},
			}},
		}, nil).Times(2),
		albMock.EXPECT().DescribeTargetHealth(gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes(),
	)
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		ALB: albMock,
		EC2: ec2Mock,
	})
	ctx := context.Background()
	result := cagecli.RollOut(ctx)
	if result.Error != nil {
		t.Fatalf(result.Error.Error())
	}
}
func TestCage_RollOut3(t *testing.T) {
	// canary taskがtgに登録されない場合は打ち切る
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	envars.serviceDefinition = ReadServiceDefinition("fixtures/service.json")
	ctrl := gomock.NewController(t)
	mocker, ecsMock, _, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")
	albMock := mock_elbv2iface.NewMockELBV2API(ctrl)
	albMock.EXPECT().RegisterTargets(gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
	albMock.EXPECT().DeregisterTargets(gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()
	albMock.EXPECT().WaitUntilTargetDeregistered(gomock.Any()).Return(nil).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
		TargetHealthDescriptions: []*elbv2.TargetHealthDescription{{
			Target: &elbv2.TargetDescription{
				Id:               aws.String("192.0.0.1"),
				Port:             aws.Int64(8000),
				AvailabilityZone: aws.String("us-west-2"),
			},
			TargetHealth: &elbv2.TargetHealth{
				State: aws.String("healthy"),
			},
		}, {
			Target: &elbv2.TargetDescription{
				Id:               aws.String("127.0.0.1"),
				Port:             aws.Int64(8000),
				AvailabilityZone: aws.String("us-west-2"),
			},
			TargetHealth: &elbv2.TargetHealth{
				State: aws.String("unused"),
			},
		}},
	}, nil).AnyTimes()
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		EC2: ec2Mock,
		ALB: albMock,
	})
	ctx := context.Background()
	result := cagecli.RollOut(ctx)
	assert.NotNil(t, result.Error)
}

func TestCage_StartGradualRollOut5(t *testing.T) {
	// lbがないサービスの場合もロールアウトする
	envars := DefaultEnvars()
	newTimer = fakeTimer
	defer recoverTimer()
	envars.serviceDefinition = ReadServiceDefinition("fixtures/service.json")
	envars.serviceDefinition.LoadBalancers = nil
	ctrl := gomock.NewController(t)
	_, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, 2, "FARGATE")
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		EC2: ec2Mock,
		ALB: albMock,
	})
	ctx := context.Background()
	if res := cagecli.RollOut(ctx); res.Error != nil {
		t.Fatalf(res.Error.Error())
	} else if res.ServiceIntact {
		t.Fatalf("no")
	}
}

func TestCage_RollOut_EC2(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	for _, v := range []int64{1, 2, 15} {
		log.Info("====")
		canaryInstanceArn := "arn:aws:ecs:us-west-2:1234567689012:container-instance/abcdefg-hijk-lmn-opqrstuvwxyz"
		attributeValue := "true"
		envars := DefaultEnvars()
		envars.CanaryInstanceArn = canaryInstanceArn
		ctrl := gomock.NewController(t)
		mctx, ecsMock, albMock, ec2Mock := Setup(ctrl, envars, v, "ec2")
		ecsMock.EXPECT().ListAttributes(gomock.Any()).Return(&ecs.ListAttributesOutput{
			Attributes: []*ecs.Attribute{
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
		if taskCnt := mctx.TaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		cagecli := NewCage(&Input{
			Env: envars,
			ECS: ecsMock,
			EC2: ec2Mock,
			ALB: albMock,
		})
		ctx := context.Background()
		result := cagecli.RollOut(ctx)
		if result.Error != nil {
			t.Fatalf("%s", result.Error)
		}
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, int64(1), mctx.ServiceSize())
		assert.Equal(t, v, mctx.TaskSize())
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
	if taskCnt := mctx.TaskSize(); taskCnt != 1 {
		t.Fatalf("current tasks not setup: %d/%d", 1, taskCnt)
	}
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		EC2: ec2Mock,
		ALB: albMock,
	})
	ctx := context.Background()
	result := cagecli.RollOut(ctx)
	if result.Error == nil {
		t.Fatal("Rollout with no container instance should be error")
	} else {
		assert.True(t, regexp.MustCompile("canaryInstanceArn is required").MatchString(result.Error.Error()))
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
	if taskCnt := mctx.TaskSize(); taskCnt != 1 {
		t.Fatalf("current tasks not setup: %d/%d", 1, taskCnt)
	}
	ecsMock.EXPECT().ListAttributes(gomock.Any()).Return(&ecs.ListAttributesOutput{
		Attributes: []*ecs.Attribute{},
	}, nil).AnyTimes()
	ecsMock.EXPECT().PutAttributes(gomock.Any()).Return(&ecs.PutAttributesOutput{}, nil).AnyTimes()
	cagecli := NewCage(&Input{
		Env: envars,
		ECS: ecsMock,
		EC2: ec2Mock,
		ALB: albMock,
	})
	ctx := context.Background()
	result := cagecli.RollOut(ctx)
	if result.Error != nil {
		t.Fatalf("%s", result.Error)
	}
	assert.False(t, result.ServiceIntact)
	assert.Equal(t, int64(1), mctx.ServiceSize())
	assert.Equal(t, int64(1), mctx.TaskSize())
}

func TestCage_CreateNextTaskDefinition(t *testing.T) {
	envars := &Envars{
		TaskDefinitionArn: "arn://task",
	}
	ctrl := gomock.NewController(t)
	e := mock_ecsiface.NewMockECSAPI(ctrl)
	e.EXPECT().DescribeTaskDefinition(gomock.Any()).Return(
		&ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: &ecs.TaskDefinition{TaskDefinitionArn: aws.String("arn://task")},
		}, nil)
	// nextTaskDefinitionArnがある場合はdescribeTaskDefinitionから返す
	cagecli := &cage{env: envars, ecs: e}
	o, err := cagecli.CreateNextTaskDefinition()
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, envars.TaskDefinitionArn, *o.TaskDefinitionArn)
}
