package commands

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

func readString() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\n")
}

func readUint() uint32 {
	line := readString()
	number, err := strconv.Atoi(line)
	if err != nil {
		log.Fatal("Invalid value")
	}

	return uint32(number)
}
