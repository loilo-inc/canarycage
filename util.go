package cage

import (
	"errors"
	"fmt"
	"github.com/apex/log"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func ReadFileAndApplyEnvars(path string) ([]byte, error) {
	if d, err := ioutil.ReadFile(path); err != nil {
		return nil, err
	} else {
		str := string(d)
		reg := regexp.MustCompile("\\${(.+?)}")
		submatches := reg.FindAllStringSubmatch(str, -1)
		for _, m := range submatches {
			if envar, ok := os.LookupEnv(m[1]); ok {
				str = strings.Replace(str, m[0], envar, -1)
			} else {
				log.Warnf("envar literal '%s' found in %s but was not defined. filled by empty string", m[0], path)
				str = strings.Replace(str, m[0], "", -1)
			}
		}
		return []byte(str), nil
	}
}

func NewErrorf(f string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(f, args...))
}
