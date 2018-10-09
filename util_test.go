package cage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_ecs"
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"os"
	"log"
)

func TestExtractAlbId(t *testing.T) {
	if out, err := ExtractAlbId("arn:aws:elasticloadbalancing:us-west-2:1111:loadbalancer/app/alb/12345"); err != nil {
		t.Fatalf(err.Error())
	} else {
		exp := "app/alb/12345"
		if out != exp {
			t.Fatalf("expected: %s, but got: %s", exp, out)
		}
	}
	if _, err := ExtractAlbId("hogehoge"); err == nil {
		t.Fatalf("should return error if alb is invalid")
	}
}
func TestExtractTargetGroupId(t *testing.T) {
	if out, err := ExtractTargetGroupId("arn:aws:elasticloadbalancing:us-west-2:1111:targetgroup/tg/12345"); err != nil {
		t.Fatalf(err.Error())
	} else {
		exp := "targetgroup/tg/12345"
		if out != exp {
			t.Fatalf("expected: %s, but got: %s", exp, out)
		}
	}
	if _, err := ExtractTargetGroupId("hoge"); err == nil {
		t.Fatalf("should return error if tg is invalid")
	}
}

func TestEstimateRollOutCount(t *testing.T) {
	assert.Equal(t, int64(1), EstimateRollOutCount(1))
	assert.Equal(t, int64(2), EstimateRollOutCount(2))
	assert.Equal(t, int64(4), EstimateRollOutCount(10))
}

func TestEnsureReplaceCount(t *testing.T) {
	// 2^0 = 1
	assert.Equal(t, int64(1), EnsureReplaceCount(0, 0, 4))
	// 2^1 = 2
	assert.Equal(t, int64(2), EnsureReplaceCount(1, 1, 6))
	// 2^2 = 4
	assert.Equal(t, int64(4), EnsureReplaceCount(6, 2, 15))
	assert.Equal(t, int64(1), EnsureReplaceCount(14, 3, 15))
}

func TestTimeAdd(t *testing.T) {
	now := time.Now()
	after5min := now
	after5min = now.Add(time.Duration(5) * time.Minute)
	assert.Equal(t, after5min.After(now), true)
	assert.NotEqual(t, now.Unix(), after5min.Unix())
}

func TestReadFileAndApplyEnvars(t *testing.T) {
	os.Setenv("HOGE", "hogehoge")
	os.Setenv("FUGA", "fugafuga")
	d, err := ReadFileAndApplyEnvars("./fixtures/template.txt")
	if err != nil {
		t.Fatalf(err.Error())
	}
	s := string(d)
	e := `HOGE=hogehoge
FUGA=fugafuga
fugafuga=hogehoge`
	if s != e {
		log.Fatalf("e: %s, a: %s", e, s)
	}
}

func TestFindService(t *testing.T) {
	ctrl := gomock.NewController(t)
	ecsmock := mock_ecs.NewMockECSAPI(ctrl)
	ecsmock.EXPECT().ListServices(gomock.Any()).Return(&ecs.ListServicesOutput{
		ServiceArns: aws.StringSlice([]string {
			"arn:aws:ecs:us-west-2:828165198279:service/lns-share-ee78c92",
			"arn:aws:ecs:us-west-2:828165198279:service/lns-web-ee78c92",
		}),
	}, nil)
	o, err := FindService(ecsmock, aws.String("hgoe"), "lns-share")
	assert.Nil(t, err)
	assert.Equal(t, "lns-share-ee78c92", o)
}

func TestFindService2(t *testing.T) {
	ctrl := gomock.NewController(t)
	ecsmock := mock_ecs.NewMockECSAPI(ctrl)
	ecsmock.EXPECT().ListServices(gomock.Any()).Return(&ecs.ListServicesOutput{
		ServiceArns: aws.StringSlice([]string {
			"arn:aws:ecs:us-west-2:828165198279:service/lns-share-ee78c92",
			"arn:aws:ecs:us-west-2:828165198279:service/lns-share-ee78c93",
			"arn:aws:ecs:us-west-2:828165198279:service/lns-web-ee78c92",
		}),
	}, nil)
	_, err := FindService(ecsmock, aws.String("hgoe"), "lns-share")
	assert.NotNil(t, err)
}

func TestFindService3(t *testing.T) {
	ctrl := gomock.NewController(t)
	ecsmock := mock_ecs.NewMockECSAPI(ctrl)
	ecsmock.EXPECT().ListServices(gomock.Any()).Return(&ecs.ListServicesOutput{
		ServiceArns: aws.StringSlice([]string {
			"arn:aws:ecs:us-west-2:828165198279:service/lns-share-ee78c92",
			"arn:aws:ecs:us-west-2:828165198279:service/lns-share-ee78c93",
			"arn:aws:ecs:us-west-2:828165198279:service/lns-web-ee78c92",
		}),
	}, nil)
	_, err := FindService(ecsmock, aws.String("hgoe"), "lns-hgoe")
	assert.NotNil(t, err)
}