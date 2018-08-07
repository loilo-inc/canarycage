package main

import (
	"io/ioutil"
	"github.com/apex/log"
	"encoding/base64"
	"testing"
)

func TestUnmarshalTaskDefinition(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/task-definition-current.json")
	log.SetLevel(log.DebugLevel)
	out, _ := UnmarshalTaskDefinition(base64.StdEncoding.EncodeToString(d))
	if *out.Family != "service:1" {
		t.Fatalf("e: %s, but: %s", "canarycage:1", *out.Family)
	}

}

func TestUnmarshalServiceDefinition(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/service-definition-current.json")
	log.SetLevel(log.DebugLevel)
	out, err := UnmarshalServiceDefinition(base64.StdEncoding.EncodeToString(d))
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	log.Debugf("%f", *out.TaskDefinition)
	if *out.TaskDefinition != "service:1" {
		t.Fatalf("e: service:1, a: %s", *out.TaskDefinition)
	}
	if *out.ServiceName != "service-current" {
		t.Fatalf("e: %s, a: %s", "next-service", *out.ServiceName)
	}
}