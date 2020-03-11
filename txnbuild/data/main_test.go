package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManageDataMemoRequired(t *testing.T) {
	tt := assert.New(t)
	manageData := NewManageDataMemoRequired(true)
	tt.NoError(manageData.Validate())
	tt.Equal("config.memo_required", manageData.Name)
	tt.Equal([]byte("1"), manageData.Value)

	manageData = NewManageDataMemoRequired(false)
	tt.NoError(manageData.Validate())
	tt.Equal("config.memo_required", manageData.Name)
	tt.Equal([]byte("0"), manageData.Value)
}
