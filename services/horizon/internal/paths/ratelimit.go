package paths

import (
	"context"

	"golang.org/x/time/rate"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var (
	// ErrLimitExceeded indicates that the in memory order book is not yet populated
	ErrLimitExceeded = errors.New("Empty orderbook")
)

// RateLimitedFinder is a Finder implementation which limits the number of path finding requests.
type RateLimitedFinder struct {
	finder  Finder
	limiter *rate.Limiter
}

// NewRateLimitedFinder constructs a new RateLimitedFinder which enforces a per
// second limit on path finding requests.
func NewRateLimitedFinder(finder Finder, limit int) *RateLimitedFinder {
	return &RateLimitedFinder{
		finder:  finder,
		limiter: rate.NewLimiter(rate.Limit(limit), limit),
	}
}

// Limit returns the per second limit of path finding requests.
func (f *RateLimitedFinder) Limit() int {
	return f.limiter.Burst()
}

// Find implements the Finder interface and returns ErrLimitExceeded if the
// RateLimitedFinder is unable to complete the request due to rate limits.
func (f *RateLimitedFinder) Find(ctx context.Context, q Query, maxLength uint) ([]Path, uint32, error) {
	if !f.limiter.Allow() {
		return nil, 0, ErrLimitExceeded
	}
	return f.finder.Find(ctx, q, maxLength)
}

// FindFixedPaths implements the Finder interface and returns ErrLimitExceeded if the
// RateLimitedFinder is unable to complete the request due to rate limits.
func (f *RateLimitedFinder) FindFixedPaths(
	ctx context.Context,
	sourceAsset xdr.Asset,
	amountToSpend xdr.Int64,
	destinationAssets []xdr.Asset,
	maxLength uint,
) ([]Path, uint32, error) {
	if !f.limiter.Allow() {
		return nil, 0, ErrLimitExceeded
	}
	return f.finder.FindFixedPaths(ctx, sourceAsset, amountToSpend, destinationAssets, maxLength)
}
