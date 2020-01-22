package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigOption_UsageText(t *testing.T) {
	configOpt := ConfigOption{
		Usage:  "Port to listen and serve on",
		EnvVar: "PORT",
	}
	assert.Equal(t, "Port to listen and serve on (PORT)", configOpt.UsageText())
}
