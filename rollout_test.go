package cage

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"testing"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_ecs"
	"github.com/loilo-inc/canarycage/mock/mock_cloudwatch"
	"github.com/apex/log"
	"github.com/loilo-inc/canarycage/test"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"sync"
	"github.com/loilo-inc/canarycage/mock/mock_elbv2"
)

const kCurrentServiceName = "service-current"
const kNextServiceName = "service-next"

func DefaultEnvars() *Envars {
	d, _ := ioutil.ReadFile("fixtures/task-definition.json")
	o := base64.StdEncoding.EncodeToString(d)
	return &Envars{
		Region:                   aws.String("us-west-2"),
		RollOutPeriod:            aws.Int64(0),
		Cluster:                  aws.String("cage-test"),
		CurrentServiceName:       aws.String(kCurrentServiceName),
		NextServiceName:          aws.String(kNextServiceName),
		NextTaskDefinitionBase64: aws.String(o),
		AvailabilityThreshold:    aws.Float64(0.9970),
		ResponseTimeThreshold:    aws.Float64(1),
	}
}

func (envars *Envars) Setup(ctrl *gomock.Controller, currentTaskCount int64) (*test.MockContext, *Context) {
	ecsMock := mock_ecs.NewMockECSAPI(ctrl)
	cwMock := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
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
	cwMock.EXPECT().GetMetricStatistics(gomock.Any()).DoAndReturn(mocker.GetMetricStatics).AnyTimes()
	albMock.EXPECT().DescribeTargetGroups(gomock.Any()).DoAndReturn(mocker.DescribeTargetGroups).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes()
	o, _ := base64.StdEncoding.DecodeString(*envars.NextTaskDefinitionBase64)
	var register *ecs.RegisterTaskDefinitionInput
	_ = json.Unmarshal(o, register)
	td, _ := mocker.RegisterTaskDefinition(register)
	a := &ecs.CreateServiceInput{
		ServiceName: envars.CurrentServiceName,
		LoadBalancers: []*ecs.LoadBalancer{
			{
				TargetGroupArn: aws.String("arn://aaa/hoge/targetgroup/aaa/bbb"),
				ContainerName:  aws.String("container"),
				ContainerPort:  aws.Int64(80),
			},
		},
		TaskDefinition: td.TaskDefinition.TaskDefinitionArn,
		DesiredCount:   aws.Int64(currentTaskCount),
	}
	_, _ = mocker.CreateService(a)
	return mocker, &Context{
		Ecs: ecsMock,
		Cw:  cwMock,
		Alb: albMock,
	}
}

func TestEnvars_StartGradualRollOut(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	for _, v := range []int64{1, 2, 15} {
		log.Info("====")
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		mctx, ctx := envars.Setup(ctrl, v)
		if mctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := mctx.TaskSize(); taskCnt != v {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		result, err := envars.StartGradualRollOut(ctx)
		if err != nil {
			t.Fatalf("%s", err)
		}
		assert.Nil(t, result.HandledError)
		assert.False(t,  *result.Rolledback)
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
	envars.NextServiceDefinitionBase64 = aws.String(base64.StdEncoding.EncodeToString(d))
	ctrl := gomock.NewController(t)
	mocker, ctx := envars.Setup(ctrl, 2)
	result, err := envars.StartGradualRollOut(ctx)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Nil(t, result.HandledError)
	assert.False(t, *result.Rolledback)
	assert.Equal(t, int64(1), mocker.ServiceSize())
	assert.Equal(t, int64(2), mocker.TaskSize())
}

func TestEnvars_StartGradualRollOut3(t *testing.T) {
	// カナリアテストに失敗した場合ロールバックすることを確かめる
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	ctrl := gomock.NewController(t)
	mocker, ctx := envars.Setup(ctrl, 4)
	cwMock := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	ctx.Cw = cwMock
	m := make(map[string]int)
	mux := sync.Mutex{}
	cwMock.EXPECT().GetMetricStatistics(gomock.Any()).DoAndReturn(func(input *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
		o := func() *cloudwatch.GetMetricStatisticsOutput {
			var ret = &cloudwatch.Datapoint{}
			mux.Lock()
			defer mux.Unlock()
			cnt, ok := m[*input.MetricName]
			if !ok {
				cnt = 0
			}
			switch *input.MetricName {
			case "RequestCount":
				sum := 10000.0
				ret.Sum = &sum
			case "HTTPCode_ELB_5XX_Count":
				sum := 1.0
				ret.Sum = &sum
			case "HTTPCode_Target_5XX_Count":
				sum := 100.0
				if cnt == 0 {
					sum = 1.0
				}
				ret.Sum = &sum
			case "TargetResponseTime":
				average := 0.11
				ret.Average = &average
			}
			m[*input.MetricName] = cnt + 1
			return &cloudwatch.GetMetricStatisticsOutput{
				Datapoints: []*cloudwatch.Datapoint{ret},
			}
		}()
		return o, nil
	}).AnyTimes()
	result, err := envars.StartGradualRollOut(ctx)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.NotNil(t, result.HandledError)
	assert.True(t, *result.Rolledback)
	assert.Equal(t, int64(1), mocker.ServiceSize())
	if _, ok := mocker.GetService("service-current"); !ok {
		t.Fatalf("service-current doesn't exists")
	}
	if _, ok := mocker.GetService("service-next"); ok {
		t.Fatalf("service-next still exists")
	}
	assert.Equal(t, int64(4), mocker.TaskSize())
}

func TestEnvars_Rollback(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	newTimer = fakeTimer
	defer recoverTimer()
	envars := DefaultEnvars()
	ctrl := gomock.NewController(t)
	mocker, ctx := envars.Setup(ctrl, 2)
	mocker.CreateService(&ecs.CreateServiceInput{
		Cluster:        envars.Cluster,
		ServiceName:    envars.NextServiceName,
		TaskDefinition: aws.String("arn://current"),
		DesiredCount:   aws.Int64(12),
	})
	log.Debugf("%d", mocker.ServiceSize())
	err := envars.Rollback(ctx, aws.Int64(10), aws.String("hoge"))
	if err != nil {
		t.Fatal(err.Error())
	}
	if mocker.ServiceSize() != 1 {
		t.Fatal("next service still exists")
	}
	o, _ := ctx.Ecs.ListTasks(&ecs.ListTasksInput{
		ServiceName: envars.CurrentServiceName,
	})
	if l := len(o.TaskArns); l != 10 {
		t.Fatalf("next service was not rollbacked: E: %d, A: %d", 10, l)
	}
}

func TestEnvars_CreateNextTaskDefinition(t *testing.T) {
	envars := &Envars{
		NextTaskDefinitionArn: aws.String("arn://task"),
	}
	ctrl := gomock.NewController(t)
	e := mock_ecs.NewMockECSAPI(ctrl)
	e.EXPECT().DescribeTaskDefinition(gomock.Any()).Return(
		&ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: &ecs.TaskDefinition{TaskDefinitionArn: aws.String("arn://task"),},
		}, nil)
	// nextTaskDefinitionArnがある場合はdescribeTaskDefinitionから返す
	o, err := envars.CreateNextTaskDefinition(e)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, *envars.NextTaskDefinitionArn, *o.TaskDefinitionArn)
}

func TestEnvars_CreateNextTaskDefinition2(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/service.json")
	j := base64.StdEncoding.EncodeToString(d)
	envars := &Envars{
		NextTaskDefinitionBase64: aws.String(j),
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
