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
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"net/http"
	"time"
	"golang.org/x/sync/errgroup"
)

const kCurrentServiceName = "itg-test-service-current"
const kNextServiceName = "itg-test-service-next"
const kHealthyTDArn = "cage-test-server-healthy:16"
const kUnhealthyTDArn = "cage-test-server-unhealthy:16"
const kUpButBuggyTDArn = "cage-test-server-up-but-buggy:16"
const kUpButSlowTDArn = "cage-test-server-up-but-slow:16"
const kUpAndExitTDArn = "cage-test-server-up-and-exit:16"
const kUpButExitTDArn = "cage-test-server-up-but-exit:16"

func SetupAws() (*ecs.ECS, *cloudwatch.CloudWatch, *elbv2.ELBV2) {
	ses, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	return ecs.New(ses), cloudwatch.New(ses), elbv2.New(ses)
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
		input.ServiceName = aws.String(kCurrentServiceName)
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

func setupEnvars(envars *cage.Envars) {
	if d, err := ioutil.ReadFile("cage.json"); err != nil {
		log.Fatalf(err.Error())
	} else if err := json.Unmarshal(d, envars); err != nil {
		log.Fatalf(err.Error())
	}
}

func GetAlbDNS(alb elbv2iface.ELBV2API, arn *string) (*string, error) {
	out, err := alb.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{arn},
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
	alb *elbv2.ELBV2,
	interval time.Duration,
	stop chan bool,
) error {
	dns, err := GetAlbDNS(alb, envars.LoadBalancerArn)
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
				defer resp.Body.Close()
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

func TestHealthyToHealthy(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	envars := &cage.Envars{}
	setupEnvars(envars)
	envars.NextTaskDefinitionArn = aws.String(kHealthyTDArn)
	envars.CurrentServiceName = aws.String(kCurrentServiceName)
	envars.NextServiceName = aws.String(kNextServiceName)
	if err := cage.EnsureEnvars(envars); err != nil {
		t.Fatalf(err.Error())
	}
	ec, cw, alb := SetupAws()
	if err := ensureCurrentService(ec, envars); err != nil {
		t.Fatalf(err.Error())
	}
	defer cleanupService(ec, envars, envars.CurrentServiceName)
	if err := cleanupService(ec, envars, envars.NextServiceName); err != nil {
		t.Fatalf(err.Error())
	}
	stop := make(chan bool)
	eg := errgroup.Group{}
	eg.Go(func() error {
		ret := envars.StartGradualRollOut(ec, cw)
		stop <- true
		return ret
	})
	eg.Go(func() error {
		return PollLoadBalancer(envars, alb, time.Duration(10)*time.Second, stop)
	})
	err := eg.Wait()
	if err != nil {
		t.Fatalf(err.Error())
	}
}
