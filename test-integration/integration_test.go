package test_integration

import (
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/loilo-inc/canarycage"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
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
	log.Infof("checking if service %s exists", *envars.Service)
	if o, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.Service},
	}); err != nil {
		return err
	} else if len(o.Services) == 0 || *o.Services[0].Status == "INACTIVE" {
		log.Infof("%s", o.Failures)
		log.Infof("service %s doesn't exist", *envars.Service)
		input.ServiceName = envars.Service
		input.TaskDefinition = aws.String(kHealthyTDArn)
		log.Infof("creating service '%s'", *input.ServiceName)
		if _, err := awsEcs.CreateService(input); err != nil {
			return err
		}
	} else {
		log.Infof("service '%s' exists. ensure desiredCount=%d", *envars.Service, *input.DesiredCount)
		if _, err := awsEcs.UpdateService(&ecs.UpdateServiceInput{
			Cluster:      envars.Cluster,
			Service:      o.Services[0].ServiceName,
			DesiredCount: input.DesiredCount,
		}); err != nil {
			return err
		}
	}
	log.Infof("waiting for service '%s' become STABLE", *envars.Service)
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.Service},
	}); err != nil {
		return err
	}
	log.Infof("service '%s' ensured. now %d tasks running", *envars.Service)
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

func setupEnvars(
	service string,
	tdArn string,
) *cage.Envars {
	envars := &cage.Envars{
		Region:  aws.String("us-east-2"),
		Cluster: aws.String("cage-test"),
		Service: &service,
		TaskDefinitionArn: &tdArn,
	}
	cage.EnsureEnvars(envars)
	return envars
}

func testInternal(t *testing.T, envars *cage.Envars) (*cage.RollOutResult) {
	if err := cage.EnsureEnvars(envars); err != nil {
		t.Fatalf(err.Error())
	}
	ctx := setup()
	if err := ensureCurrentService(ctx.Ecs, envars); err != nil {
		t.Fatalf(err.Error())
	}
	if err := cleanupService(ctx.Ecs, envars, envars.CanaryService); err != nil {
		t.Fatalf(err.Error())
	}
	return envars.RollOut(ctx)
}


func testAbnormal(t *testing.T, tdarn string, servicePostfix string) {
	log.SetLevel(log.InfoLevel)
	ctx := setup()
	envars := setupEnvars(
		fmt.Sprintf("%s-%s", "service", servicePostfix),
		tdarn,
	)
	defer func() {
		cleanupService(ctx.Ecs, envars, envars.Service)
		cleanupService(ctx.Ecs, envars, envars.CanaryService)
	}()
	result := testInternal(t, envars)
	assert.True(t, result.ServiceIntact)
	assert.NotNil(t, result.Error)
	o, _ := ctx.Ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.Service, envars.CanaryService},
	})
	assert.Equal(t, int64(2), *o.Services[0].DesiredCount)
	//assert.Equal(t, "INACTIVE", *o.Services[1].Status)
}

func TestHealthyToHealthy(t *testing.T) {
	ctx := setup()
	envars := setupEnvars("service-healthy2healthy", kHealthyTDArn)
	defer func() {
		cleanupService(ctx.Ecs, envars, envars.Service)
		cleanupService(ctx.Ecs, envars, envars.CanaryService)
	}()
	result := testInternal(t, envars)
	assert.Nil(t, result.Error)
	assert.False(t, result.ServiceIntact)
}

func TestHealthyToNotUp(t *testing.T) {
	// 新規サービスのタスクが起動しない場合もロールバックされること
	// waitServicesStableを使いきるので600*2sec程度かかる
	testAbnormal(t, kUpButExitTDArn, "healthy2exit")
}

func TestHealthyToUnHealthy(t *testing.T) {
	// 新規サービスのタスクがALBヘルスチェック通らない場合ロールバックされること
	testAbnormal(t, kUnhealthyTDArn, "healthy2unhealthy")
}
