package commands

import (
	"bufio"
	"fmt"
	"github.com/howeyc/gopass"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)
var out io.Writer = os.Stdout

func readString(prompt string, private bool) (string, error) {
	fmt.Fprintf(os.Stdout, prompt)
	var line string
	var err error

	if private {
		str, err := gopass.GetPasswdMasked()
		if err != nil {
			return "", err
		}
		line = string(str)
	} else {
		line, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
	}
	return strings.Trim(line, "\n"), nil
}

func readUint() uint32 {
	line, err := readString("", false)
	if err != nil {
		panic(err)
	}
	number, err := strconv.Atoi(line)
	if err != nil {
		log.Fatal("Invalid value")
	}

	return uint32(number)
}

func printf(format string, a ...interface{}) {
	fmt.Fprintf(out, format, a...)
}

func println(a ...interface{}) {
	fmt.Fprintln(out, a...)
}

func getPassword(prompt string) (pass string) {
	pass, err := readString(prompt, true)
	if err != nil {
		panic(err)
	}
	return
}

func savePassword() (pass string) {
	pass = getPassword("Enter your password (leave empty if none): ")
	if pass != "" {
		confirm := getPassword("Confirm your password: ")
		if pass != confirm {
			fmt.Println("Passwords do not match. Please try again.")
			savePassword()
		}
	}
	return
}
