package commands

import (
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/crypto/derivation"
	"github.com/stellar/go/keypair"
	"github.com/tyler-smith/go-bip39"
)

var wordsRegexp = regexp.MustCompile(`^[a-z]+$`)
var count, startID uint32

var allowedNumbers = map[uint32]bool{12: true, 15: true, 18: true, 21: true, 24: true}

var AccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Display accounts for a given mnemonic code",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("How many words? ")
		wordsCount := readUint()
		if _, exist := allowedNumbers[wordsCount]; !exist {
			log.Fatal("Invalid value, allowed values: 12, 15, 18, 21, 24")
		}

		words := make([]string, wordsCount)
		for i := uint32(0); i < wordsCount; i++ {
			fmt.Printf("Enter word #%-4d", i+1)
			words[i] = readString()
			if !wordsRegexp.MatchString(words[i]) {
				fmt.Println("Invalid word, try again.")
				i--
			}
		}

		fmt.Printf("Enter password (leave empty if none): ")
		password := readString()

		mnemonic := strings.Join(words, " ")
		fmt.Println("Mnemonic:", mnemonic)

		seed, err := bip39.NewSeedWithErrorChecking(mnemonic, password)
		if err != nil {
			log.Fatal("Invalid words or checksum")
		}

		fmt.Println("BIP39 Seed:", hex.EncodeToString(seed))

		masterKey, err := derivation.DeriveForPath(derivation.StellarAccountPrefix, seed)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("m/44'/148' key:", hex.EncodeToString(masterKey.Key))

		fmt.Println("")

		for i := uint32(startID); i < startID+count; i++ {
			key, err := masterKey.Derive(derivation.FirstHardenedIndex + i)
			if err != nil {
				log.Fatal(err)
			}

			kp, err := keypair.FromRawSeed(key.RawSeed())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(fmt.Sprintf(derivation.StellarAccountPathFormat, i), kp.Address(), kp.Seed())
		}
	},
}

func init() {
	AccountsCmd.Flags().Uint32VarP(&count, "count", "c", 10, "number of accounts to display")
	AccountsCmd.Flags().Uint32VarP(&startID, "start", "s", 0, "ID of the first wallet to display")
}
