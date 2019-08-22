// +build ignore

package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/vfsgen"
)

func main() {
	fs := filter.Keep(http.Dir("assets"), func(path string, fi os.FileInfo) bool {
		return fi.IsDir() || strings.HasSuffix(path, ".sql")
	})
	if err := vfsgen.Generate(fs, vfsgen.Options{
		PackageName: "scenarios",
	}); err != nil {
		log.Fatalln(err)
	}
}
