// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockS3 struct {
	mock.Mock
	s3iface.S3API
}

func TestWriteACLRuleOverride(t *testing.T) {

	mockS3 := &MockS3{}
	s3Storage := S3Storage{
		ctx:              context.Background(),
		svc:              mockS3,
		bucket:           "bucket",
		prefix:           "prefix",
		unsignedRequests: false,
		writeACLrule:     s3.ObjectCannedACLBucketOwnerFullControl,
	}

	aclRule := s3Storage.GetACLWriteRule()
	assert.Equal(t, aclRule, s3.ObjectCannedACLBucketOwnerFullControl)
}

func TestWriteACLRuleDefault(t *testing.T) {

	mockS3 := &MockS3{}
	s3Storage := S3Storage{
		ctx:              context.Background(),
		svc:              mockS3,
		bucket:           "bucket",
		prefix:           "prefix",
		unsignedRequests: false,
		writeACLrule:     "",
	}

	aclRule := s3Storage.GetACLWriteRule()
	assert.Equal(t, aclRule, s3.ObjectCannedACLPublicRead)
}
