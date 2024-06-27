package env_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/loilo-inc/canarycage/env"
)

func TestTimeAdd(t *testing.T) {
	now := time.Now()
	after5min := now
	after5min = now.Add(time.Duration(5) * time.Minute)
	assert.Equal(t, after5min.After(now), true)
	assert.NotEqual(t, now.Unix(), after5min.Unix())
}

func TestReadFileAndApplyEnvars(t *testing.T) {
	os.Setenv("HOGE", "hogehoge")
	os.Setenv("FUGA", "fugafuga")
	d, err := env.ReadFileAndApplyEnvars("./fixtures/template.txt")
	if err != nil {
		t.Fatalf(err.Error())
	}
	s := string(d)
	e := `HOGE=hogehoge
FUGA=fugafuga
fugafuga=hogehoge`
	if s != e {
		log.Fatalf("e: %s, a: %s", e, s)
	}
}
