package main

import (
	"testing"
	"io/ioutil"
	"encoding/base64"
	"github.com/apex/log"
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
}

func TestUnmarshalServiceDefinition(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/service-definition.json")
	log.SetLevel(log.DebugLevel)
	out, err := UnmarshalServiceDefinition(base64.StdEncoding.EncodeToString(d))
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	log.Debugf("%f", *out.TaskDefinition)
	if (*out.TaskDefinition) != "family:1" {
		t.Fatalf("expected family:1, but: %s", *out.TaskDefinition)
	}
}