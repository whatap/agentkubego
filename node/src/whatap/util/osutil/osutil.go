package osutil

import (
	"os"
	"strings"
)

func GetEnv(variable string, def string) string {
	val := os.Getenv(variable)
	if len(val) == 0 {
		val = def
	}

	return val
}

func GetEnvAll(h2 func(string, string)) {

	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if len(pair) > 1 {
			h2(pair[0], pair[1])
		}
	}
	return
}

func Hostname(h1 func(string)) (ret error) {
	name, err := os.Hostname()
	if err == nil {
		h1(name)
	}
	ret = err

	return
}
