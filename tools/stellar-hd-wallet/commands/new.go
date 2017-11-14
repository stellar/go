package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip39"
)

var NewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generates a new mnemonic code",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			log.Fatal(err)
		}

		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			log.Fatal(err)
		}

		words := strings.Split(mnemonic, " ")
		for i := 0; i < len(words); i++ {
			fmt.Printf("word %02d/24: %10s", i+1, words[i])
			readString()
		}

		fmt.Println("WARNING! Store the words above in a save place!")
		fmt.Println("WARNING! If you lose your words, you will lose access to funds in all derived accounts!")
		fmt.Println("WARNING! Anyone who has access to these words can spend your funds!")
		fmt.Println("")
		fmt.Println("Use: `stellar-hd-wallet accounts` command to see generated accounts.")
	},
}
