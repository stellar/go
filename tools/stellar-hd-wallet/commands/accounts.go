package commands

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stellar/go/tools/stellar-hd-wallet/derive"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

var wordsRegexp = regexp.MustCompile(`^[a-z]+$`)
var count, startID uint32

var AccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Display accounts for a given mnemonic code",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("How many words? ")
		wordsCount := readUint()
		if wordsCount < 12 {
			log.Fatal("Invalid value (min 12)")
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

		seed := bip39.NewSeed(mnemonic, password)
		masterKey, err := bip32.NewMasterKey(seed)
		if err != nil {
			log.Fatal(err)
		}

		paths, keypairs, err := derive.GetKeyPairs(masterKey, startID, count)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("")
		for i, keypair := range keypairs {
			fmt.Printf("%s %s %s\n", paths[i], keypair.Address(), keypair.Seed())
		}
	},
}

func init() {
	AccountsCmd.Flags().Uint32VarP(&count, "count", "c", 10, "number of accounts to display")
	AccountsCmd.Flags().Uint32VarP(&startID, "start", "s", 0, "ID of the first wallet to display")
}
