package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBucketRegex(t *testing.T) {
	const bucketFullPath = "prd/core-live/core-live-001/bucket/88/af/31/bucket-88af31f4c51afe5ea75861642359376feb623de5bec4354fa56ab752aeec8f36.xdr.gz"
	const bucketPath = "bucket/88/af/31/bucket-88af31f4c51afe5ea75861642359376feb623de5bec4354fa56ab752aeec8f36.xdr.gz"

	bp, e := getBucketPath(bucketRegex, bucketFullPath)
	if !assert.NoError(t, e) {
		return
	}

	assert.Equal(t, bucketPath, bp)
}
