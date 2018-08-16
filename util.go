package main

import (
	"regexp"
	"fmt"
	"math"
	"errors"
)

func ExtractAlbId(arn string) (string, error) {
	regex := regexp.MustCompile(`^.+(app/.+?)$`)
	if group := regex.FindStringSubmatch(arn); group == nil || len(group) == 1 {
		return "", errors.New(fmt.Sprintf("could not find alb id in '%s'", arn))
	} else {
		return group[1], nil
	}
}

func ExtractTargetGroupId(arn string) (string, error) {
	regex := regexp.MustCompile(`^.+(targetgroup/.+?)$`)
	if group := regex.FindStringSubmatch(arn); group == nil || len(group) == 1 {
		return "", errors.New(fmt.Sprintf("could not find target group id in '%s'", arn))
	} else {
		return group[1], nil
	}
}

func EstimateRollOutCount(originalTaskCount int64) int64 {
	var i int64 = 0
	for ; int64(math.Pow(2, float64(i)))-1 < originalTaskCount; i++ {
	}
	return i
}

func EnsureReplaceCount(
	totalReplacedCount int64,
	totalRollOutCount int64,
	originalCount int64,
) (int64) {
	return int64(math.Min(
		math.Pow(2, float64(totalRollOutCount)),
		float64(originalCount-totalReplacedCount)),
	)
}

type StringPtrWrap struct {
	value *string
	dummy *string
}

func NewStringPtrWrap() *StringPtrWrap {
	dummy := ""
	return &StringPtrWrap{
		value: &dummy,
		dummy: &dummy,
	}
}

func (s *StringPtrWrap) Unwrap() *string {
	if *s.dummy == *s.value {
		return nil
	}
	return s.value
}