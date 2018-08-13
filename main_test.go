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
	// 初回はdesired countだけ減らす
	if a, r := EnsureReplaceCount(2, 0, 0, 4); a != 0 {
		t.Fatalf("E: %d, A: %d", 0, a)
	} else if r != 2 {
		t.Fatalf("E: %d, A: %d", 2, r)
	}
	// 二回目以降はceil(log2(DesiredCount))
	if a, r := EnsureReplaceCount(2, 1, 1, 6); a != 4 {
		t.Fatalf("E: %d, A: %d", 4, a)
	} else if r != 4 {
		t.Fatalf("E: %d, A: %d", 4, r)
	}
	// 三回目以降はoriginal countになるまで
	if a, r := EnsureReplaceCount(2, 6, 2, 15); a != 8 {
		t.Fatalf("E: %d, A: %d", 8, a)
	} else if r != 8 {
		t.Fatalf("E: %d, A: %d", 8, r)
	}
	if a, r := EnsureReplaceCount(2, 14, 3, 15); a != 1 {
		t.Fatalf("E: %d, A: %d", 1, a)
	} else if r != 1 {
		t.Fatalf("E: %d, A: %d", 1, r)
	}
}
