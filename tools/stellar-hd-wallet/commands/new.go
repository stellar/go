package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip39"
)

var wordsCount uint32

var NewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generates a new mnemonic code",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		var entSize int
		switch wordsCount {
		case 12:
			entSize = 128
		case 15:
			entSize = 160
		case 18:
			entSize = 192
		case 21:
			entSize = 224
		case 24:
			entSize = 256
		default:
			log.Fatal("`words` param invalid")
		}

		entropy, err := bip39.NewEntropy(entSize)
		if err != nil {
			log.Fatal(err)
		}

		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			log.Fatal(err)
		}

		words := strings.Split(mnemonic, " ")
		for i := uint32(0); i < wordsCount; i++ {
			fmt.Printf("#%-5d %10s", i+1, words[i])
			readString()
		}

		fmt.Println("WARNING! Store the words above in a save place!")
		fmt.Println("WARNING! If you lose your words, you will lose access to funds in all derived accounts!")
		fmt.Println("WARNING! Anyone who has access to these words can spend your funds!")
	},
}

func init() {
	NewCmd.Flags().Uint32VarP(&wordsCount, "words", "w", 24, "number of words to generate")
}
