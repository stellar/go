package config

import (
	"github.com/stellar/go/keypair"
)

type Config struct {
	Port                           int             `valid:"required"`
	UsingProxy                     bool            `valid:"optional" toml:"using_proxy"`
	Bitcoin                        *bitcoinConfig  `valid:"optional" toml:"bitcoin"`
	Ethereum                       *ethereumConfig `valid:"optional" toml:"ethereum"`
	AccessControlAllowOriginHeader string          `valid:"optional" toml:"access_control_allow_origin_header"`

	Stellar struct {
		Horizon           string `valid:"required" toml:"horizon"`
		NetworkPassphrase string `valid:"required" toml:"network_passphrase"`
		// TokenAssetCode is asset code of token that will be purchased using BTC or ETH.
		TokenAssetCode string `valid:"required" toml:"token_asset_code"`
		// NeedsAuthorize should be set to true if issuers's authorization required flag is set.
		NeedsAuthorize bool `valid:"optional" toml:"needs_authorize"`
		// IssuerPublicKey is public key of the assets issuer.
		IssuerPublicKey string `valid:"required,stellar_accountid" toml:"issuer_public_key"`
		// DistributionPublicKey is public key of the distribution account.
		// Distribution account can be the same account as issuer account however it's recommended
		// to use a separate account.
		// Distribution account is also used to fund new accounts.
		DistributionPublicKey string `valid:"required,stellar_accountid" toml:"distribution_public_key"`
		// SignerSecretKey is:
		// * Distribution's secret key if only one instance of Bifrost is deployed.
		// * Channel's secret key of Distribution account if more than one instance of Bifrost is deployed.
		// https://www.stellar.org/developers/guides/channels.html
		// Signer's sequence number will be consumed in transaction's sequence number.
		SignerSecretKey string `valid:"required,stellar_seed" toml:"signer_secret_key"`
		// StartingBalance is the starting amount of XLM for newly created accounts.
		// Default value is 2.1. Increase it if you need Data records / other custom entities on new account.
		StartingBalance string `valid:"optional,stellar_amount" toml:"starting_balance"`
		// LockUnixTimestamp defines unix timestamp when user account will be unlocked.
		LockUnixTimestamp uint64 `valid:"optional" toml:"lock_unix_timestamp"`
	} `valid:"required" toml:"stellar"`
	Database struct {
		Type string `valid:"matches(^postgres$)"`
		DSN  string `valid:"required"`
	} `valid:"required"`
}

type bitcoinConfig struct {
	MasterPublicKey string `valid:"required" toml:"master_public_key"`
	// Minimum value of transaction accepted by Bifrost in BTC.
	// Everything below will be ignored.
	MinimumValueBtc string `valid:"required" toml:"minimum_value_btc"`
	// TokenPrice is a price of one token in BTC
	TokenPrice string `valid:"required" toml:"token_price"`
	// Host only
	RpcServer string `valid:"required" toml:"rpc_server"`
	RpcUser   string `valid:"optional" toml:"rpc_user"`
	RpcPass   string `valid:"optional" toml:"rpc_pass"`
	Testnet   bool   `valid:"optional" toml:"testnet"`
}

type ethereumConfig struct {
	NetworkID       string `valid:"required,int" toml:"network_id"`
	MasterPublicKey string `valid:"required" toml:"master_public_key"`
	// Minimum value of transaction accepted by Bifrost in ETH.
	// Everything below will be ignored.
	MinimumValueEth string `valid:"required" toml:"minimum_value_eth"`
	// TokenPrice is a price of one token in ETH
	TokenPrice string `valid:"required" toml:"token_price"`
	// Host only
	RpcServer string `valid:"required" toml:"rpc_server"`
}

func (c Config) SignerPublicKey() string {
	kp := keypair.MustParse(c.Stellar.SignerSecretKey)
	return kp.Address()
}
