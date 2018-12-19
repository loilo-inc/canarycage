package cage

import (
	"encoding/base64"
	"encoding/json"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_ecs"
	"github.com/loilo-inc/canarycage/mock/mock_elbv2"
	"github.com/loilo-inc/canarycage/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func DefaultEnvars() *Envars {
	d, _ := ioutil.ReadFile("fixtures/task-definition.json")
	o := base64.StdEncoding.EncodeToString(d)
	return &Envars{
		Region:               aws.String("us-west-2"),
		Cluster:              aws.String("cage-test"),
		Service:              aws.String("service"),
		CanaryService:        aws.String("service-canary"),
		TaskDefinitionBase64: &o,
	}
}

func (envars *Envars) Setup(ctrl *gomock.Controller, currentTaskCount int64, launchType string) (*test.MockContext, *Context) {
	ecsMock := mock_ecs.NewMockECSAPI(ctrl)
	albMock := mock_elbv2.NewMockELBV2API(ctrl)
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
	albMock.EXPECT().DescribeTargetGroups(gomock.Any()).DoAndReturn(mocker.DescribeTargetGroups).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes()
	albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any()).DoAndReturn(mocker.DescribeTargetGroupAttibutes).AnyTimes()
	o, _ := base64.StdEncoding.DecodeString(*envars.TaskDefinitionBase64)
	var register *ecs.RegisterTaskDefinitionInput
	_ = json.Unmarshal(o, register)
	td, _ := mocker.RegisterTaskDefinition(register)
	a := &ecs.CreateServiceInput{
		ServiceName: envars.Service,
		LoadBalancers: []*ecs.LoadBalancer{
			{
				TargetGroupArn: aws.String("arn://aaa/hoge/targetgroup/aaa/bbb"),
				ContainerName:  aws.String("container"),
				ContainerPort:  aws.Int64(80),
			},
		},
		TaskDefinition: td.TaskDefinition.TaskDefinitionArn,
		DesiredCount:   aws.Int64(currentTaskCount),
		LaunchType:     aws.String(launchType),
	}
	_, _ = mocker.CreateService(a)
	return mocker, &Context{
		Ecs: ecsMock,
		Alb: albMock,
	}
}

func TestEnvars_RollOut(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	for _, v := range []int64{1, 2, 15} {
		log.Info("====")
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mctx, ctx := envars.Setup(ctrl, v, "FARGATE")
		if mctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := mctx.TaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		result := envars.RollOut(ctx)
		if result.Error != nil {
			t.Fatalf("%s", result.Error)
		}
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, int64(1), mctx.ServiceSize())
		assert.Equal(t, v, mctx.TaskSize())
	}
}

func TestEnvars_StartGradualRollOut2(t *testing.T) {
	// service definitionのjsonから読み込む
	log.SetLevel(log.InfoLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	d, _ := ioutil.ReadFile("fixtures/service.json")
	envars.ServiceDefinitionBase64 = aws.String(base64.StdEncoding.EncodeToString(d))
	ctrl := gomock.NewController(t)
	mocker, ctx := envars.Setup(ctrl, 2, "FARGATE")
	result := envars.RollOut(ctx)
	if result.Error != nil {
		t.Fatalf(result.Error.Error())
	}
	assert.False(t, result.ServiceIntact)
	assert.Equal(t, int64(1), mocker.ServiceSize())
	assert.Equal(t, int64(2), mocker.TaskSize())
}

func TestEnvars_RollOut2(t *testing.T) {
	// canary taskがtgに登録されるまで少し待つ
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	d, _ := ioutil.ReadFile("fixtures/service.json")
	envars.ServiceDefinitionBase64 = aws.String(base64.StdEncoding.EncodeToString(d))
	ctrl := gomock.NewController(t)
	mocker, ctx := envars.Setup(ctrl, 2, "FARGATE")
	albMock := mock_elbv2.NewMockELBV2API(ctrl)
	gomock.InOrder(
		albMock.EXPECT().DescribeTargetHealth(gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
			TargetHealthDescriptions: []*elbv2.TargetHealthDescription{{
				Target: &elbv2.TargetDescription{
					Id:               aws.String("127.0.0.1"),
					Port:             aws.Int64(8000),
					AvailabilityZone: aws.String("us-west-2"),
				},
				TargetHealth: &elbv2.TargetHealth{
					State: aws.String("unused"),
				},
			}},
		}, nil).Times(2),
		albMock.EXPECT().DescribeTargetHealth(gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes(),
	)
	ctx.Alb = albMock
	result := envars.RollOut(ctx)
	if result.Error != nil {
		t.Fatalf(result.Error.Error())
	}
}
func TestEnvars_RollOut3(t *testing.T) {
	// canary taskがtgに登録されない場合は打ち切る
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	d, _ := ioutil.ReadFile("fixtures/service.json")
	envars.ServiceDefinitionBase64 = aws.String(base64.StdEncoding.EncodeToString(d))
	ctrl := gomock.NewController(t)
	_, ctx := envars.Setup(ctrl, 2, "FARGATE")
	albMock := mock_elbv2.NewMockELBV2API(ctrl)
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
	ctx.Alb = albMock
	result := envars.RollOut(ctx)
	assert.NotNil(t, result.Error)
}
func TestEnvars_StartGradualRollOut5(t *testing.T) {
	// lbがないサービスの場合もロールアウトする
	envars := DefaultEnvars()
	d, _ := ioutil.ReadFile("fixtures/service.json")
	input := &ecs.CreateServiceInput{}
	_ = json.Unmarshal(d, input)
	input.LoadBalancers = nil
	newTimer = fakeTimer
	defer recoverTimer()
	o, _ := json.Marshal(input)
	envars.ServiceDefinitionBase64 = aws.String(base64.StdEncoding.EncodeToString(o))
	ctrl := gomock.NewController(t)
	_, ctx := envars.Setup(ctrl, 2, "FARGATE")
	if res := envars.RollOut(ctx); res.Error != nil {
		t.Fatalf(res.Error.Error())
	} else if res.ServiceIntact {
		t.Fatalf("no")
	}
}
func TestEnvars_RollOut_EC2(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	for _, v := range []int64{1, 2, 15} {
		log.Info("====")
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mctx, ctx := envars.Setup(ctrl, v, "EC2")
		if mctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := mctx.TaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		result := envars.RollOut(ctx)
		if result.Error != nil {
			t.Fatalf("%s", result.Error)
		}
		assert.False(t, result.ServiceIntact)
		assert.Equal(t, int64(1), mctx.ServiceSize())
		assert.Equal(t, v, mctx.TaskSize())
	}
}

func TestEnvars_CreateNextTaskDefinition(t *testing.T) {
	envars := &Envars{
		TaskDefinitionArn: aws.String("arn://task"),
	}
	ctrl := gomock.NewController(t)
	e := mock_ecs.NewMockECSAPI(ctrl)
	e.EXPECT().DescribeTaskDefinition(gomock.Any()).Return(
		&ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: &ecs.TaskDefinition{TaskDefinitionArn: aws.String("arn://task")},
		}, nil)
	// nextTaskDefinitionArnがある場合はdescribeTaskDefinitionから返す
	o, err := envars.CreateNextTaskDefinition(e)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, *envars.TaskDefinitionArn, *o.TaskDefinitionArn)
}

func TestEnvars_CreateNextTaskDefinition2(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/service.json")
	j := base64.StdEncoding.EncodeToString(d)
	envars := &Envars{
		TaskDefinitionBase64: aws.String(j),
	}
	ctrl := gomock.NewController(t)
	e := mock_ecs.NewMockECSAPI(ctrl)
	e.EXPECT().RegisterTaskDefinition(gomock.Any()).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{TaskDefinitionArn: aws.String("arn://next")},
	}, nil)
	// nextTaskDefinitionBase64がある場合は新規作成
	o, err := envars.CreateNextTaskDefinition(e)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, "arn://next", *o.TaskDefinitionArn)
}
