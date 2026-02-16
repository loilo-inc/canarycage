package prompt_test

import (
	"strings"
	"testing"

	"github.com/loilo-inc/canarycage/v5/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/stretchr/testify/assert"
)

func TestPrompter(t *testing.T) {
	t.Run("Confirm", func(t *testing.T) {
		t.Run("yes", func(t *testing.T) {
			reader := strings.NewReader("yes\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.Confirm("test", "yes")
			assert.NoError(t, err)
		})
		t.Run("no", func(t *testing.T) {
			reader := strings.NewReader("no\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.Confirm("test", "yes")
			assert.Error(t, err)
		})
	})
	envars := &env.Envars{
		Region:  "ap-northeast-1",
		Cluster: "test-cluster",
		Service: "test-service",
	}
	t.Run("ConfirmTask", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			reader := strings.NewReader("ap-northeast-1\ntest-cluster\nyes\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.ConfirmTask(envars)
			assert.NoError(t, err)
		})
	})
	t.Run("ConfirmService", func(t *testing.T) {
		t.Run("yes", func(t *testing.T) {
			reader := strings.NewReader("ap-northeast-1\ntest-cluster\ntest-service\nyes\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.ConfirmService(envars)
			assert.NoError(t, err)
		})
		t.Run("region mismatch", func(t *testing.T) {
			reader := strings.NewReader("us-west-2\ntest-cluster\nyes\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.ConfirmTask(envars)
			assert.Error(t, err)
		})
		t.Run("cluster mismatch", func(t *testing.T) {
			reader := strings.NewReader("ap-northeast-1\ndifferent-cluster\nyes\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.ConfirmTask(envars)
			assert.Error(t, err)
		})
		t.Run("service mismatch", func(t *testing.T) {
			reader := strings.NewReader("ap-northeast-1\ntest-cluster\ndifferent-service\nyes\n")
			prompter := prompt.NewPrompter(reader)
			err := prompter.ConfirmTask(envars)
			assert.Error(t, err)
		})
	})
}
