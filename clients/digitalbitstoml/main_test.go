package digitalbitstoml

import "log"

// ExampleGetTOML gets the digitalbits.toml file for livenet.digitalbits.io
func ExampleClient_GetDigitalBitsToml() {
	_, err := DefaultClient.GetDigitalBitsToml("livenet.digitalbits.io")
	if err != nil {
		log.Fatal(err)
	}
}
