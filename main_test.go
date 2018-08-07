package main

import (
	"testing"
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

func TestEstimateRollOutCount(t *testing.T) {
	arr := [][]int{{1, 1, 1}, {2, 1, 2}, {10, 2, 4}}
	for _, v := range arr {
		o := EstimateRollOutCount(v[0], v[1])
		if o != v[2] {
			t.Fatalf("E: %d, A: %d: originalCount=%d, nextDisiredCount=%d", v[2], o, v[0], v[1])
		}
	}
}

func TestEnsureReplaceCount(t *testing.T) {
	if a := EnsureReplaceCount(1, 0, 2); a != 2 {
		t.Fatalf("E: %d, A: %d", 2, a)
	}
	if a := EnsureReplaceCount(3, 4, 15); a != 8 {
		t.Fatalf("E: %d, A: %d", 8, a)
	}
	if a := EnsureReplaceCount(4, 14, 16); a != 2 {
		t.Fatalf("E: %d, A: %d", 2, a)
	}
}