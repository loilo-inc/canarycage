package main

import "testing"

func TestExtractAlbId(t *testing.T) {
	if out, err := ExtractAlbId("arn:aws:elasticloadbalancing:us-west-2:1111:loadbalancer/app/alb/12345"); err != nil {
		t.Fatalf(err.Error())
	} else {
		exp := "app/alb/12345"
		if out !=  exp{
			t.Fatalf("expected: %s, but got: %s", exp, out)
		}
	}
}
func TestExtractTargetGroupId(t *testing.T) {
	if out, err := ExtractTargetGroupId("arn:aws:elasticloadbalancing:us-west-2:1111:targetgroup/tg/12345"); err != nil {
		t.Fatalf(err.Error())
	} else {
		exp := "targetgroup/tg/12345"
		if out !=  exp{
			t.Fatalf("expected: %s, but got: %s", exp, out)
		}
	}
}
