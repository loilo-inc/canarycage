package cage

import (
	"os"
	"regexp"
	"strings"

	"github.com/apex/log"
)

func ReadFileAndApplyEnvars(path string) ([]byte, error) {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	str := string(d)
	reg := regexp.MustCompile(`\${(.+?)}`)
	submatches := reg.FindAllStringSubmatch(str, -1)
	for _, m := range submatches {
		if envar, ok := os.LookupEnv(m[1]); ok {
			str = strings.Replace(str, m[0], envar, -1)
		} else {
			log.Fatalf("envar literal '%s' found in %s but was not defined", m[0], path)
		}
	}
	return []byte(str), nil
}
