package cage

import (
	"github.com/apex/log"
	"time"
	"math"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"golang.org/x/sync/errgroup"
	"errors"
)

type ServiceHealth struct {
	availability float64
	responseTime float64
}

type NoDataPointsFoundError struct {
	Input *cloudwatch.GetMetricStatisticsInput
}

func (v *NoDataPointsFoundError) Error() string {
	return ""
}

func (envars *Envars) GetServiceMetricStatistics(
	cw cloudwatchiface.CloudWatchAPI,
	lbId string,
	tgId string,
	metricName string,
	unit string,
	startTime time.Time,
	endTime time.Time,
) (float64, error) {
	log.Infof(
		"getStatistics: LoadBalancer=%s, TargetGroup=%s, metricName=%s, unit=%s",
		lbId, tgId, metricName, unit,
	)
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace: aws.String("AWS/ApplicationELB"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("LoadBalancer"),
				Value: aws.String(lbId),
			}, {
				Name:  aws.String("TargetGroup"),
				Value: aws.String(tgId),
			},
		},
		Statistics: []*string{&unit},
		MetricName: &metricName,
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     envars.RollOutPeriod,
	}
	out, err := cw.GetMetricStatistics(input)
	if err != nil {
		log.Errorf("failed to get CloudWatch's '%s' metric statistics due to: %s", metricName, err.Error())
		return 0, err
	}
	if (metricName == "RequestCount" || metricName ==  "TargetResponseTime") && len(out.Datapoints) == 0 {
		return 0, &NoDataPointsFoundError{Input: input}
	}
	var ret float64 = 0
	switch unit {
	case "Sum":
		for _, v := range out.Datapoints {
			ret += *v.Sum
		}
	case "Average":
		for _, v := range out.Datapoints {
			ret += *v.Average
		}
		ret /= float64(len(out.Datapoints))
	default:
		err = NewErrorf("unsupported unit type: %s", unit)
	}
	return ret, err
}

func (envars *Envars) AccumulatePeriodicServiceHealth(
	cw cloudwatchiface.CloudWatchAPI,
	targetGroupArn *string,
	startTime time.Time,
	endTime time.Time,
) (*ServiceHealth, error) {
	var (
		lbId string
		tgId string
		err  error
	)
	if lbId, err = ExtractAlbId(*envars.LoadBalancerArn); err != nil {
		return nil, err
	}
	if tgId, err = ExtractTargetGroupId(*targetGroupArn); err != nil {
		return nil, err
	}
	maxRetryCnt := 20 // 15sec x 20 = 5min
	for i := 0; i < maxRetryCnt; i++ {
		eg := errgroup.Group{}
		requestCnt := 0.0
		elb5xxCnt := 0.0
		target5xxCnt := 0.0
		responseTime := 0.0
		accumulate := func(metricName string, unit string, dest *float64) func() (error) {
			return func() (error) {
				v, err := envars.GetServiceMetricStatistics(cw, lbId, tgId, metricName, unit, startTime, endTime)
				if err == nil {
					*dest = v
				}
				return err
			}
		}
		eg.Go(accumulate("RequestCount", "Sum", &requestCnt))
		eg.Go(accumulate("HTTPCode_ELB_5XX_Count", "Sum", &elb5xxCnt))
		eg.Go(accumulate("HTTPCode_Target_5XX_Count", "Sum", &target5xxCnt))
		eg.Go(accumulate("TargetResponseTime", "Average", &responseTime))
		err := eg.Wait()
		if err != nil {
			switch err.(type) {
			case *NoDataPointsFoundError:
				// タイミングによってCloudWatchのメトリクスデータポイントがまだ存在しない場合がある
				log.Warnf(
					"no data points found on CloudWatch Metrics between %s ~ %s. will retry after %d seconds",
					startTime.String(), endTime.String(), 15,
				)
			default:
				log.Errorf("failed to accumulate periodic service health due to: %s", err.Error())
				return nil, err
			}
		} else {
			if requestCnt == 0 && elb5xxCnt == 0 {
				return nil, errors.New("failed to get precise metric data")
			} else {
				avl := (requestCnt - target5xxCnt) / (requestCnt + elb5xxCnt)
				avl = math.Max(0, math.Min(1, avl))
				return &ServiceHealth{
					availability: avl,
					responseTime: responseTime,
				}, nil
			}
		}
		<-newTimer(time.Duration(15) * time.Second).C
	}
	return nil, NewErrorf("no data points found in 20 retries")
}
