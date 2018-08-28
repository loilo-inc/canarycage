package cage

import (
	"testing"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestEnvars_GetServiceMetricStatistics(t *testing.T) {
	envars := &Envars{}
	ctrl := gomock.NewController(t)
	cw := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	cw.EXPECT().GetMetricStatistics(gomock.Any()).Return(&cloudwatch.GetMetricStatisticsOutput{
		Datapoints: []*cloudwatch.Datapoint{
			{
				Average: aws.Float64(5.5),
			},
		},
	}, nil)
	o, err := envars.GetServiceMetricStatistics(
		cw, "lb", "tg", "dummy", "Average", time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, 5.5, o)
}

func TestEnvars_GetServiceMetricStatistics2(t *testing.T) {
	envars := &Envars{}
	ctrl := gomock.NewController(t)
	cw := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	cw.EXPECT().GetMetricStatistics(gomock.Any()).Return(&cloudwatch.GetMetricStatisticsOutput{
		Datapoints: []*cloudwatch.Datapoint{},
	}, nil).AnyTimes()
	// datapointsがない場合はNoDataPointsFoundErrorを出す
	_, err := envars.GetServiceMetricStatistics(
		cw, "lb", "tg", "RequestCount", "Average", time.Now(), time.Now(),
	)
	assert.NotNil(t, err)
	_, ok := err.(*NoDataPointsFoundError)
	assert.True(t, ok)
}
func TestEnvars_AccumulatePeriodicServiceHealth2(t *testing.T) {
	defer func() { newTimer = time.NewTimer }()
	newTimer = fakeTimer
	envars := &Envars{
	}
	ctrl := gomock.NewController(t)
	cw := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	callCnt := 0
	cw.EXPECT().GetMetricStatistics(gomock.Any()).DoAndReturn(func(input *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
		if callCnt <= 3 {
			callCnt++
			return &cloudwatch.GetMetricStatisticsOutput{
				Datapoints: []*cloudwatch.Datapoint{},
			}, nil
		}
		dp := &cloudwatch.Datapoint{}
		switch *input.MetricName {
		case "RequestCount":
			dp.Sum = aws.Float64(10000)
		case "HTTPCode_Target_5XX_Count":
			dp.Sum = aws.Float64(0)
		case "HTTPCode_ELB_5XX_Count":
			dp.Sum = aws.Float64(0)
		case "TargetResponseTime":
			dp.Average = aws.Float64(0.5)
		}
		return &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []*cloudwatch.Datapoint{dp},
		}, nil
	}).AnyTimes()
	// cwがdata pointsを返さなくても指定範囲内でリトライする
	o, err := envars.AccumulatePeriodicServiceHealth(
		cw, aws.String("hoge/app/aa/bb"), aws.String("hoge/targetgroup/aa/bb"), time.Now(), time.Now(),
	)
	assert.Nil(t, err)
	assert.Equal(t, 1.0, o.availability)
	assert.Equal(t, 0.5, o.responseTime)
}

func TestEnvars_AccumulatePeriodicServiceHealth(t *testing.T) {
	envars := &Envars{}
	ctrl := gomock.NewController(t)
	cw := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	cw.EXPECT().GetMetricStatistics(gomock.Any()).Return(&cloudwatch.GetMetricStatisticsOutput{
		Datapoints: []*cloudwatch.Datapoint{},
	}, nil).AnyTimes()
	defer func() {
		newTimer = time.NewTimer
	}()
	callCnt := 0
	newTimer = func(d time.Duration) *time.Timer {
		callCnt++
		return fakeTimer(d)
	}
	_, err := envars.AccumulatePeriodicServiceHealth(
		cw, aws.String("hoge/app/aa/bb"), aws.String("hoge/targetgroup/aa/bb"), time.Now(), time.Now(),
	)
	assert.NotNil(t, err)
	assert.Equal(t, 20, callCnt)
}
