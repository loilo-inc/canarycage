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

func EstimateRollOutCount(originalTaskCount int) int {
	var i = 0
	for ; int(math.Pow(2, float64(i)))-1 < originalTaskCount; i++ {
	}
	return i
}

func EnsureReplaceCount(
	totalReplacedCount int,
	totalRollOutCount int,
	originalCount int,
) int {
	// DesiredCount以下のカナリア追加は意味がないので2回目以降はこの指数より上を使う
	return int(math.Min(
		math.Pow(2, float64(totalRollOutCount)),
		float64(originalCount-totalReplacedCount)),
	)
}

