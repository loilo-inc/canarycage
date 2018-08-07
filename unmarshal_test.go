package main

import (
	"io/ioutil"
	"github.com/apex/log"
	"encoding/base64"
	"testing"
)

func TestUnmarshalTaskDefinition(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/task-definition.json")
	log.SetLevel(log.DebugLevel)
	out, _ := UnmarshalTaskDefinition(base64.StdEncoding.EncodeToString(d))
	if *out.Family != "canarycage:1" {
		t.Fatalf("e: %s, but: %s", "canarycage:1", *out.Family)
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
	if *out.TaskDefinition != "family:1" {
		t.Fatalf("expected family:1, but: %s", *out.TaskDefinition)
	}
	if *out.ServiceName != "next-service" {
		t.Fatalf("e: %s, a: %s", "next-service", *out.ServiceName)
	}
}