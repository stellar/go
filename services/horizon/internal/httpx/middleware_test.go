package httpx

import (
	"testing"
)

func TestMiddlewareSanitizesRoutesForPrometheus(t *testing.T) {
	for _, setup := range []struct {
		name     string
		route    string
		expected string
	}{
		{
			"normal routes are unaffected",
			"/accounts",
			"/accounts",
		},
		{
			"named regexes are replaced with their name",
			"/accounts/{account_id:\\w+}/effects",
			"/accounts/account_id/effects",
		},
		{
			"unnamed regexes are removed",
			"/accounts/{\\w+}/effects",
			"/accounts//effects",
		},
	} {
		t.Run(setup.name, func(t *testing.T) {
			result := routeRegexp.ReplaceAllString(setup.route, "$2")
			if result != setup.expected {
				t.Errorf("Expected %q to be sanitized to %q, but got %q", setup.route, setup.expected, result)
			}
		})
	}

}
