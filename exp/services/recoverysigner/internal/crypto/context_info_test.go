package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextInfo(t *testing.T) {
	const accountAddress = "GAEGZCXGFXBSI77L2YQ6WFGLGZYM3Q6USE2YII5JQFIWVJUGALHIYLJK"
	const signingAddress = "GBJLCILAGXNXRU7BFNOQZG3AYN4CRSTQLKU42MLII7XJW4LR56SY6Y6T"

	contextInfo := ContextInfo(accountAddress, signingAddress)

	wantContextInfo := []byte("GAEGZCXGFXBSI77L2YQ6WFGLGZYM3Q6USE2YII5JQFIWVJUGALHIYLJK,GBJLCILAGXNXRU7BFNOQZG3AYN4CRSTQLKU42MLII7XJW4LR56SY6Y6T")
	assert.Equal(t, wantContextInfo, contextInfo)
}
