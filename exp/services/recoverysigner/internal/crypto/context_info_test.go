package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextInfo(t *testing.T) {
	const accountAddress = "GAEGZCXGFXBSI77L2YQ6WFGLGZYM3Q6USE2YII5JQFIWVJUGALHIYLJK"

	contextInfo := ContextInfo(accountAddress)

	wantContextInfo := []byte("GAEGZCXGFXBSI77L2YQ6WFGLGZYM3Q6USE2YII5JQFIWVJUGALHIYLJK")
	assert.Equal(t, wantContextInfo, contextInfo)
}
