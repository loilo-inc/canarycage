package test_integration

import (
	"io/ioutil"
	"github.com/aws/aws-sdk-go/service/ecs"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/aws"
	"testing"
	"github.com/loilo-inc/canarycage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"net/http"
	"time"
	"golang.org/x/sync/errgroup"
	"github.com/stretchr/testify/assert"
	"fmt"
)

const kCurrentServiceName = "itg-test-service-current"
const kNextServiceName = "itg-test-service-next"
const kHealthyTDArn = "cage-test-server-healthy:16"
const kUnhealthyTDArn = "cage-test-server-unhealthy:15"
const kUpButBuggyTDArn = "cage-test-server-up-but-buggy:15"
const kUpButSlowTDArn = "cage-test-server-up-but-slow:15"
const kUpAndExitTDArn = "cage-test-server-up-and-exit:15"
const kUpButExitTDArn = "cage-test-server-up-but-exit:15"

func setup() (*cage.Context) {
	ses, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	return &cage.Context{
		Ecs: ecs.New(ses),
		Cw:  cloudwatch.New(ses),
		Alb: elbv2.New(ses),
	}
}

func ensureCurrentService(awsEcs ecsiface.ECSAPI, envars *cage.Envars) (error) {
	d, err := ioutil.ReadFile("service-template.json")
	if err != nil {
		return err
	}
	input := &ecs.CreateServiceInput{}
	if err := json.Unmarshal(d, input); err != nil {
		return err
	}
	log.Infof("checking if service %s exists", *envars.CurrentServiceName)
	if o, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CurrentServiceName},
	}); err != nil {
		return err
	} else if len(o.Services) == 0 || *o.Services[0].Status == "INACTIVE" {
		log.Infof("%s", o.Failures)
		log.Infof("service %s doesn't exist", *envars.CurrentServiceName)
		input.ServiceName = envars.CurrentServiceName
		input.TaskDefinition = aws.String(kHealthyTDArn)
		log.Infof("creating service '%s'", *input.ServiceName)
		if _, err := awsEcs.CreateService(input); err != nil {
			return err
		}
	} else {
		log.Infof("service '%s' exists. ensure desiredCount=%d", *envars.CurrentServiceName, *input.DesiredCount)
		if _, err := awsEcs.UpdateService(&ecs.UpdateServiceInput{
			Cluster:      envars.Cluster,
			Service:      o.Services[0].ServiceName,
			DesiredCount: input.DesiredCount,
		}); err != nil {
			return err
		}
	}
	log.Infof("waiting for service '%s' become STABLE", *envars.CurrentServiceName)
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CurrentServiceName},
	}); err != nil {
		return err
	}
	log.Infof("service '%s' ensured. now %d tasks running", *envars.CurrentServiceName)
	return nil
}

func cleanupService(awsEcs ecsiface.ECSAPI, envars *cage.Envars, serviceName *string) (error) {
	log.Infof("cleaning up service '%s'...", *serviceName)
	if o, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{serviceName},
	}); err != nil {
		return err
	} else if len(o.Services) == 0 || *o.Services[0].Status == "INACTIVE" {
		return nil
	}
	if _, err := awsEcs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      envars.Cluster,
		Service:      serviceName,
		DesiredCount: aws.Int64(0),
	}); err != nil {
		return err
	}
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{serviceName},
	}); err != nil {
		return err
	}
	if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: envars.Cluster,
		Service: serviceName,
		Force:   aws.Bool(true),
	}); err != nil {
		return err
	}
	if err := awsEcs.WaitUntilServicesInactive(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{serviceName},
	}); err != nil {
		return err
	}
	log.Infof("cleanup completed.")
	return nil
}

func setupEnvars() *cage.Envars {
	envars := &cage.Envars{}
	if d, err := ioutil.ReadFile("cage.json"); err != nil {
		log.Fatalf(err.Error())
	} else if err := json.Unmarshal(d, envars); err != nil {
		log.Fatalf(err.Error())
	}
	return envars
}

func GetAlbDNS(ctx *cage.Context, envars *cage.Envars) (*string, error) {
	o, _ := ctx.Ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CurrentServiceName},
	})
	o2, _ := ctx.Alb.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: []*string{o.Services[0].LoadBalancers[0].TargetGroupArn},
	})
	out, err := ctx.Alb.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{o2.TargetGroups[0].LoadBalancerArns[0]},
	})
	if err != nil {
		log.Errorf("failed to get alb info due to: %s", err)
		return nil, err
	}
	lb := out.LoadBalancers[0]
	return lb.DNSName, nil
}

func PollLoadBalancer(
	envars *cage.Envars,
	ctx *cage.Context,
	interval time.Duration,
	stop chan bool,
) error {
	dns, err := GetAlbDNS(ctx, envars)
	if err != nil {
		log.Errorf(err.Error())
		return err
	}
	for {
		log.Infof("polling...")
		select {
		case <-stop:
			log.Infof("stop polling")
			return nil
		default:
			url := "http://" + *dns
			log.Infof("GET: " + url)
			err := func() error {
				resp, err := http.Get(url)
				defer func() {
					if resp != nil {
						resp.Body.Close()
					}
				}()
				return err
			}()
			if err != nil {
				log.Error(err.Error())
				return err
			}
			timer := time.NewTimer(interval)
			<-timer.C
		}
	}
	return nil
}

func testInternal(t *testing.T, envars *cage.Envars) (*cage.RollOutResult, error) {
	if err := cage.EnsureEnvars(envars); err != nil {
		t.Fatalf(err.Error())
	}
	ctx := setup()
	if err := ensureCurrentService(ctx.Ecs, envars); err != nil {
		t.Fatalf(err.Error())
	}
	if err := cleanupService(ctx.Ecs, envars, envars.NextServiceName); err != nil {
		t.Fatalf(err.Error())
	}
	stop := make(chan bool)
	eg := errgroup.Group{}
	var result *cage.RollOutResult
	eg.Go(func() error {
		r, err := envars.StartGradualRollOut(ctx)
		result = r
		stop <- true
		return err
	})
	eg.Go(func() error {
		return PollLoadBalancer(envars, ctx, time.Duration(10)*time.Second, stop)
	})
	return result, eg.Wait()
}

func TestHealthyToHealthy(t *testing.T) {
	ctx := setup()
	envars := setupEnvars()
	envars.NextTaskDefinitionArn = aws.String(kHealthyTDArn)
	envars.CurrentServiceName = aws.String(kCurrentServiceName)
	envars.NextServiceName = aws.String(kNextServiceName)
	defer cleanupService(ctx.Ecs, envars, envars.CurrentServiceName)
	result, err := testInternal(t, envars);
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Nil(t, result.HandledError)
	assert.False(t, *result.Rolledback)
}

func TestHealthyToHealthySkipCanary(t *testing.T) {
	ctx := setup()
	envars := setupEnvars()
	envars.SkipCanary = aws.Bool(true)
	envars.NextTaskDefinitionArn = aws.String(kHealthyTDArn)
	envars.CurrentServiceName = aws.String(kCurrentServiceName)
	envars.NextServiceName = aws.String(kNextServiceName)
	defer func() {
		cleanupService(ctx.Ecs, envars, envars.CurrentServiceName)
		cleanupService(ctx.Ecs, envars, envars.NextServiceName)
	}()
	result, err := testInternal(t, envars)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Nil(t, result.HandledError)
	assert.False(t, *result.Rolledback)
}

func testAbnormal(t *testing.T, tdarn string, servicePostfix string) error {
	log.SetLevel(log.InfoLevel)
	ctx := setup()
	envars := setupEnvars()
	envars.NextTaskDefinitionArn = aws.String(tdarn)
	envars.CurrentServiceName = aws.String(fmt.Sprintf("%s-%s", kCurrentServiceName, servicePostfix))
	envars.NextServiceName = aws.String(fmt.Sprintf("%s-%s", kNextServiceName, servicePostfix))
	defer cleanupService(ctx.Ecs, envars, envars.CurrentServiceName)
	result, err := testInternal(t, envars)
	if err != nil {
		return err
	}
	assert.True(t, *result.Rolledback)
	assert.NotNil(t, result.HandledError)
	o, _ := ctx.Ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CurrentServiceName, envars.NextServiceName},
	})
	assert.Equal(t, int64(2), *o.Services[0].DesiredCount)
	assert.Equal(t, "INACTIVE", *o.Services[1].Status)
	return nil
}

func TestHealthyToBuggy(t *testing.T) {
	// 新規サービスの5xxエラーが多すぎる場合ロールバックされること
	err := testAbnormal(t, kUpButBuggyTDArn, "healthy2buggy")
	assert.Nil(t, err)
}

func TestHealthyToSlow(t *testing.T) {
	// 新規サービスのタスクが遅い場合ロールバックされること
	err := testAbnormal(t, kUpButSlowTDArn, "healthy2slow")
	assert.Nil(t, err)
}

func TestHealthyToNotUp(t *testing.T) {
	// 新規サービスのタスクが起動しない場合もロールバックされること
	// waitServicesStableを使いきるので600*2sec程度かかる
	err := testAbnormal(t, kUpButExitTDArn, "healthy2exit")
	assert.Nil(t, err)
}

func TestHealthyToUnHealthy(t *testing.T) {
	// 新規サービスのタスクがALBヘルスチェック通らない場合ロールバックされること
	err := testAbnormal(t, kUnhealthyTDArn, "healthy2unhealthy")
	assert.Nil(t, err)
}
