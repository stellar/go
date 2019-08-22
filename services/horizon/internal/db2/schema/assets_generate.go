// +build ignore

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	if err := vfsgen.Generate(http.Dir("assets"), vfsgen.Options{
		PackageName: "schema",
	}); err != nil {
		log.Fatalln(err)
	}
}
