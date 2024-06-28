package task_test

import (
	"testing"

	"github.com/loilo-inc/canarycage/task"
	"github.com/stretchr/testify/assert"
)

func TestArnToId(t *testing.T) {
	arn := "arn://aaa/srv-1234"
	assert.Equal(t, "srv-1234", task.ArnToId(arn))
}
