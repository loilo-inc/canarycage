package task

import "strings"

func ArnToId(arn string) string {
	list := strings.Split(arn, "/")
	return list[len(list)-1]
}
