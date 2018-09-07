package horizon

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AccountTestSuite struct {
	suite.Suite
	account Account
}

// Initialize account with test data
func (suite *AccountTestSuite) SetupTest() {
	suite.account = Account{
		Data: map[string]string{
			"test":    "aGVsbG8=",
			"invalid": "a_*&^*",
		},
	}
}

// Should return the decoded value if the key exists
func (suite *AccountTestSuite) TestGetData() {
	decoded, err := suite.account.GetData("test")
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), string(decoded), "hello")
}

// Should return an empty slice if key doesn't exist
func (suite *AccountTestSuite) TestGetDataNonexistentKey() {
	decoded, err := suite.account.GetData("test2")
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), len(decoded), 0)
}

// Should return error slice if value is invalid
func (suite *AccountTestSuite) TestGetDataInvalid() {
	_, err := suite.account.GetData("invalid")
	assert.NotNil(suite.T(), err)
}

// Should return the decoded value if the key exists
func (suite *AccountTestSuite) TestMustGetData() {
	decoded := suite.account.MustGetData("test")
	assert.Equal(suite.T(), string(decoded), "hello")
}

// Should return an empty slice if key doesn't exist
func (suite *AccountTestSuite) TestMustGetDataNonexistentKey() {
	decoded := suite.account.MustGetData("test2")
	assert.Equal(suite.T(), len(decoded), 0)
}

// Should panic if the value is invalid
func (suite *AccountTestSuite) TestMustGetDataInvalid() {
	assert.Panics(suite.T(), func() { suite.account.MustGetData("invalid") })
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
