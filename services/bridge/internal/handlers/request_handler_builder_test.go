package handlers

import (
	"net/http"
	"strings"
	"testing"

	hc "github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/bridge/internal/config"

	"github.com/stellar/go/services/bridge/internal/mocks"
	"github.com/stellar/go/services/bridge/internal/test"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
)

func TestRequestHandlerBuilder(t *testing.T) {
	c := &config.Config{NetworkPassphrase: "Test SDF Network ; September 2015"}

	mockHorizon := new(hc.MockClient)
	mockHTTPClient := new(mocks.MockHTTPClient)
	mockTS := new(mocks.MockTransactionSubmitter)
	mockFederationResolver := new(mocks.MockFederationResolver)
	mockStellartomlResolver := new(mocks.MockStellartomlResolver)

	requestHandler := RequestHandler{
		Config:               c,
		Client:               mockHTTPClient,
		Horizon:              mockHorizon,
		TransactionSubmitter: mockTS,
		FederationResolver:   mockFederationResolver,
		StellarTomlResolver:  mockStellartomlResolver,
	}

	testServer := httptest.NewServer(t, http.HandlerFunc(requestHandler.Builder))
	defer testServer.Close()

	// When no sequence number IS NOT supplied
	data := test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "",
		"operations": [
		{
				"type": "create_account",
				"body": {
					"destination": "GCOEGO43PFSLE4K7WRZQNRO3PIOTRLKRASP32W7DSPBF65XFT4V6PSV3",
					"starting_balance": "50"
				}
		}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// Loading sequence number
	accountRequest := hc.AccountRequest{AccountID: "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5"}

	mockHorizon.On(
		"AccountDetail",
		accountRequest,
	).Return(
		hProtocol.Account{
			Sequence: "123",
		},
		nil,
	).Once()

	// it should return the correct XDR
	statusCode, response := mocks.JSONGetResponse(testServer, data)
	responseString := strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)
	expected := test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAfAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAACcQzubeWSycV+0cwbF23odOK1RBJ+9W+OTwl925Z8r5wAAAAAdzWUAAAAAAAAAAAGfjbT8AAAAQFyeCPQ8D2Sre8jtI/CLCLHOlS1Cc/8mdyyv//sacX+jG25fu3Bau8Bu++oe1S5g0wa0hEDsnSrkdGLqeIBamAE="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// When sequence number IS supplied
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
		{
				"type": "create_account",
				"body": {
					"destination": "GCOEGO43PFSLE4K7WRZQNRO3PIOTRLKRASP32W7DSPBF65XFT4V6PSV3",
					"starting_balance": "50"
				}
		}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR, sequence number should not be incremented
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)
	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAACcQzubeWSycV+0cwbF23odOK1RBJ+9W+OTwl925Z8r5wAAAAAdzWUAAAAAAAAAAAGfjbT8AAAAQDfN+i0CqFB8wCi05Is1Sm/T7BXX+93jTkkSihnfswAfRq55bCgy23oLbfXqr2I0OAB823qiQSBl/x0BVwmnQAM="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// build payment
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "payment",
					"body": {
						"destination": "GCOEGO43PFSLE4K7WRZQNRO3PIOTRLKRASP32W7DSPBF65XFT4V6PSV3",
						"amount": "100",
						"asset": {
							"code": "USD",
							"issuer": "GACETOPHMOLSZLG5IQ3D6KQDKCAAYUYTTQHIEY6IGZE4VOBDD2YY6YAO"
						}
					}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAACcQzubeWSycV+0cwbF23odOK1RBJ+9W+OTwl925Z8r5wAAAAFVU0QAAAAAAARJuedjlyys3UQ2PyoDUIAMUxOcDoJjyDZJyrgjHrGPAAAAADuaygAAAAAAAAAAAZ+NtPwAAABAXJCl0jGfTNrGam7+d10xsNDL/dTJ7R20KZ/1dE7aFRhwVjCyCfHjxuxNPuw3NOoPgFEf6E1mqCOllgnuUijACw=="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// Build Path Payment
	data = test.StringToJSONMap(`{
  "source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
  "sequence_number": "123",
  "operations": [
    {
        "type": "path_payment",
        "body": {
        	"source": "GBYLUAJBHGZMVAYCALM4ZRTAOY74NTSSULG42VKVY2EOWS5X2HFBB2VL",
        	"destination": "GCOEGO43PFSLE4K7WRZQNRO3PIOTRLKRASP32W7DSPBF65XFT4V6PSV3",
        	"destination_amount": "500",
        	"destination_asset": {
        		"code": "EUR",
        		"issuer": "GDOJMKTDLGGLROSSM5BV5MXIAQ3JZHASQFUV55WBJ45AFOUXSVVFGPTJ"
        	},
        	"send_max": "100",
        	"send_asset": {
        		"code": "USD",
        		"issuer": "GACETOPHMOLSZLG5IQ3D6KQDKCAAYUYTTQHIEY6IGZE4VOBDD2YY6YAO"
        	},
        	"path": [
        		{
	        		"code": "ABCDEFG",
	        		"issuer": "GD4RIHH2HWB4MPJN72G2VGLRPUXDODFNQG6DVU47HMSSSF3RIQ4UXALD"
	        	},
	        	{}
        	]
        }
    }
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAAHC6ASE5ssqDAgLZzMZgdj/GzlKizc1VVcaI60u30coQAAAAAgAAAAFVU0QAAAAAAARJuedjlyys3UQ2PyoDUIAMUxOcDoJjyDZJyrgjHrGPAAAAADuaygAAAAAAnEM7m3lksnFftHMGxdt6HTitUQSfvVvjk8JfduWfK+cAAAABRVVSAAAAAADclipjWYy4ulJnQ16y6AQ2nJwSgWle9sFPOgK6l5VqUwAAAAEqBfIAAAAAAgAAAAJBQkNERUZHAAAAAAAAAAAA+RQc+j2Dxj0t/o2qmXF9LjcMrYG8OtOfOyUpF3FEOUsAAAAAAAAAAAAAAAGfjbT8AAAAQNc/MiGkD5Jb0tPCXKMmOqbok86RvSsgoiT0ZS2FGKdOMRFbJLbwY1WU8Mvm7dibwqNN4jtHt8QUuyFVSDMFtAY="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// Build Set Options
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "set_options",
					"body": {
						"inflation_dest": "GBMPZVOMJ67WQBTBCVURDKTGL4557272EGQMAJCXPSMLOE63XPLL6SVA",
						"set_flags": [1, 2],
						"clear_flags": [4],
						"master_weight": 100,
						"low_threshold": 1,
						"medium_threshold": 2,
						"high_threshold": 3,
						"home_domain": "stellar.org",
						"signer": {
							"public_key": "GA6VMJJQM2QBPPIXK2UVTAOS4XSSSAKSCOGFQE55IMRBQR65GIVDTTQV",
							"weight": 5
						}
					}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABQAAAAEAAAAAWPzVzE+/aAZhFWkRqmZfO9/r+iGgwCRXfJi3E9u71r8AAAABAAAABAAAAAEAAAADAAAAAQAAAGQAAAABAAAAAQAAAAEAAAACAAAAAQAAAAMAAAABAAAAC3N0ZWxsYXIub3JnAAAAAAEAAAAAPVYlMGagF70XVqlZgdLl5SkBUhOMWBO9QyIYR90yKjkAAAAFAAAAAAAAAAGfjbT8AAAAQB2EGy47PwAHpFTLrlOnMS8UGBu5t/eiJvkjYkrI0slHYI40ire8c7qb4LnYwRzNbdmpDpsdTgs5pbFgkhFs6Ak="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// Build Change Trust
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "change_trust",
					"body": {
						"asset": {
							"code": "USD",
							"issuer": "GCHGRVNTXAV3OXNMCSA63BUCD6AZZX6PN2542QB6GIVTXGHQ65XS35DS"
						}
					}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABgAAAAFVU0QAAAAAAI5o1bO4K7ddrBSB7YaCH4Gc389uu81APjIrO5jw928tf/////////8AAAAAAAAAAZ+NtPwAAABAK+I2L3qHdbLVBolvVMvfZg8+YIwiafzT07tz9LlUSxBjgHDPRshDAr5tlZyuBwLUmYgsawpnGRk6knd/M1AYAg=="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// Build Allow Trust
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "allow_trust",
					"body": {
						"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
						"asset_code": "USDUSD",
						"trustor": "GBLH67TQHRNRLERQEIQJDNBV2DSWPHAPP43MBIF7DVKA7X55APUNS4LL",
						"authorize": true
					}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
	}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAAGySS3ZylffFaVZqZD6lNCUjCizHz7MLPwkN7Mxh4XN5AAAABwAAAABWf35wPFsVkjAiIJG0NdDlZ5wPfzbAoL8dVA/fvQPo2QAAAAJVU0RVU0QAAAAAAAAAAAABAAAAAAAAAAGfjbT8AAAAQFkTsXEiwbx9OPmaEw/mL+O17/spqYvaQtaWFvofSSlCPcnOPRUC1jN7L4cjsRfpsgkL7f5G+8q55T1S5VkrwAg="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// Build Account Merge
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "account_merge",
					"body": {
						"destination": "GBLH67TQHRNRLERQEIQJDNBV2DSWPHAPP43MBIF7DVKA7X55APUNS4LL"
					}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACAAAAABWf35wPFsVkjAiIJG0NdDlZ5wPfzbAoL8dVA/fvQPo2QAAAAAAAAABn420/AAAAECgtVyhN04UzzLHP3WvNAqpJnrDoT2tpa7Cy7M0dlc21BfrEiUj6at/QnbEI9K0T2OtYWpH+Yb99fTabApqmDQC"
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// Build Inflation
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "inflation",
					"body": {}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACQAAAAAAAAABn420/AAAAEA7pBom56pvvxkMp9MWR+Gv8mZVDQKJcq4V87JlXgpsbN7LNuEcBEk9pUqq8AS502G6S6xz5RU9mHteGqcCfIAO"
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	//Build Manage Data
	data = test.StringToJSONMap(`{
		"source": "GBWJES3WOKK7PRLJKZVGIPVFGQSSGCRMY7H3GCZ7BEG6ZTDB4FZXTPJ5",
		"sequence_number": "123",
		"operations": [
			{
					"type": "manage_data",
					"body": {
						"name": "test_data",
						"data": "AQIDBAUG"
					}
			}
		],
		"signers": ["SABY7FRMMJWPBTKQQ2ZN43AUJQ3Z2ZAK36VYSG2SPE2ABNQXA66H5E5G"]
		}`,
	)

	// it should return the correct XDR
	statusCode, response = mocks.JSONGetResponse(testServer, data)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)

	expected = test.StringToJSONMap(`{
		"transaction_envelope": "AAAAAgAAAABskkt2cpX3xWlWamQ+pTQlIwosx8+zCz8JDezMYeFzeQAAAGQAAAAAAAAAewAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACgAAAAl0ZXN0X2RhdGEAAAAAAAABAAAABgECAwQFBgAAAAAAAAAAAAGfjbT8AAAAQB7/h1QXlPtgs+saZXjbd7Crfiv0QhG6d+PYgSp12wc937uL8tFImTzA6QSRd8Hgq8mnE1z9CcR+O3kFJsAptQA="
		}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

}
