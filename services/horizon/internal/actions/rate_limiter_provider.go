package actions

import "github.com/throttled/throttled"

type RateLimiterProvider interface {
	GetRateLimiter() *throttled.HTTPRateLimiter
}