package config

type Config struct {
	Port     int `valid:"required"`
	Ethereum struct {
		MasterPublicKey string `valid:"required" toml:"master_public_key"`
		RpcServer       string `valid:"required" toml:"rpc_server"`
	} `valid:"required" toml:"ethereum"`
	Stellar struct {
		Horizon           string `valid:"required" toml:"horizon"`
		NetworkPassphrase string `valid:"required" toml:"network_passphrase"`
		IssuerSecretKey   string `valid:"required" toml:"issuer_secret_key"`
	} `valid:"required" toml:"stellar"`
	Database struct {
		Type string `valid:"matches(^postgres$)"`
		DSN  string `valid:"required"`
	} `valid:"required"`
}
