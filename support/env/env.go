package env

import (
	"log"
	"os"
	"strconv"
	"time"
)

// String returns the value of the environment variable "name".
// If name is not set, it returns value.
func String(name string, value string) string {
	if s := os.Getenv(name); s != "" {
		value = s
	}
	return value
}

// Int returns the value of the environment variable "name" as an int.
// If name is not set, it returns value.
func Int(name string, value int) int {
	if s := os.Getenv(name); s != "" {
		var err error
		value, err = strconv.Atoi(s)
		if err != nil {
			log.Println(name, err)
			os.Exit(1)
		}
	}
	return value
}

// Duration returns the value of the environment variable "name" as a
// time.Duration where the value of the environment variable is parsed as a
// duration string as defined in the Go stdlib time documentation. e.g. 5m30s.
// If name is not set, it returns value.
// Ref: https://golang.org/pkg/time/#ParseDuration
func Duration(name string, value time.Duration) time.Duration {
	if s := os.Getenv(name); s != "" {
		var err error
		value, err = time.ParseDuration(s)
		if err != nil {
			log.Println(name, err)
			os.Exit(1)
		}
	}
	return value
}
