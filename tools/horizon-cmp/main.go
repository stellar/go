package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	client "github.com/stellar/go/clients/horizonclient"
	protocol "github.com/stellar/go/protocols/horizon"
	cmp "github.com/stellar/go/tools/horizon-cmp/internal"
)

// maxLevels defines the maximum number of levels deep the crawler
// should go. Here's an example crawl stack:
// Level 1 = /ledgers?order=desc (finds a link to tx details)
// Level 2 = /transactions/abcdef (finds a link to a list of operations)
// Level 3 = /transactions/abcdef/operations (will not follow any links - at level 3)
const maxLevels = 3

// pathAccessLog is a regexp that gets path from ELB access log line. Example:
// 2015-05-13T23:39:43.945958Z my-loadbalancer 192.168.131.39:2817 10.0.0.1:80 0.000086 0.001048 0.001337 200 200 0 57 "GET https://www.example.com:443/transactions?order=desc HTTP/1.1" "curl/7.38.0" DHE-RSA-AES128-SHA TLSv1.2
var pathAccessLog = regexp.MustCompile(`GET https:\/\/[^/]*(/[^ ]*)`)

type pathWithLevel struct {
	Path   string
	Level  int
	Stream bool
}

func (p pathWithLevel) ID() string {
	return fmt.Sprintf("%t%s", p.Stream, p.Path)
}

var paths []pathWithLevel
var visitedPaths map[string]bool

// CLI params
var (
	horizonBase      string
	horizonTest      string
	elbAccessLogFile string
)

var rootCmd = &cobra.Command{
	Use:   "horizon-cmp",
	Short: "horizon-cmp compares two horizon servers' responses",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	visitedPaths = make(map[string]bool)

	rootCmd.Flags().StringVarP(&horizonBase, "base", "b", "", "URL of the base/old version Horizon server")
	rootCmd.Flags().StringVarP(&horizonTest, "test", "t", "", "URL of the test/new version Horizon server")
	rootCmd.Flags().StringVarP(&elbAccessLogFile, "elb-access-log-file", "a", "", "ELB access log file to replay")
}

func main() {
	rootCmd.Execute()
}

func run() {
	if horizonBase == "" || horizonTest == "" {
		fmt.Println("--base and --test params are required")
		os.Exit(1)
	}

	// Get latest ledger and operate on it's cursor to get responses at a given ledger.
	ledger := getLatestLedger()
	cursor := ledger.PagingToken()

	var accessLog *bufio.Scanner
	if elbAccessLogFile == "" {
		for _, p := range initPaths {
			paths = append(paths, pathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: false})
			paths = append(paths, pathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: true})
		}
	} else {
		file, err := os.Open(elbAccessLogFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		accessLog = bufio.NewScanner(file)
		if accessLog.Scan() {
			p := accessLog.Text()
			paths = append(paths, pathWithLevel{Path: getPathFromAccessLog(p), Level: 0, Stream: false})
			paths = append(paths, pathWithLevel{Path: getPathFromAccessLog(p), Level: 0, Stream: true})
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outputDir := fmt.Sprintf("%s/horizon-cmp-diff/%d", pwd, time.Now().Unix())

	fmt.Println("Comparing:")
	fmt.Printf("%s vs %s\n", horizonBase, horizonTest)
	fmt.Printf("[accessLog=%s ledger=%d cursor=%s outputDir=%s]\n\n", elbAccessLogFile, ledger.Sequence, cursor, outputDir)

	err = os.MkdirAll(outputDir, 0744)
	if err != nil {
		panic(err)
	}

	for len(paths) > 0 {
		var pl pathWithLevel
		pl, paths = paths[0], paths[1:]

		if pl.Level > maxLevels {
			continue
		}

		if visitedPaths[pl.ID()] {
			continue
		}

		visitedPaths[pl.ID()] = true

		fmt.Printf("[stream=%t] %s ", pl.Stream, pl.Path)

		a := cmp.NewResponse(horizonBase, pl.Path, pl.Stream)
		fmt.Print(".")
		b := cmp.NewResponse(horizonTest, pl.Path, pl.Stream)
		fmt.Print(".")

		status := ""
		if a.Equal(b) {
			status = "ok"
		} else {
			status = "diff"
		}
		fmt.Printf("%s %d %d %d\n", status, a.StatusCode, a.Size(), b.Size())
		if status == "diff" {
			a.SaveDiff(outputDir, b)
		}

		// Add new paths
		if accessLog != nil {
			if accessLog.Scan() {
				p := accessLog.Text()
				paths = append(paths, pathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: false})
				paths = append(paths, pathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: true})
			}

			if err := accessLog.Err(); err != nil {
				panic("Invalid input: " + err.Error())
			}
		} else {
			addPathsFromResponse(a, pl.Level+1)
		}
	}
}

func getLatestLedger() protocol.Ledger {
	horizon := client.Client{
		HorizonURL: horizonBase,
		HTTP:       http.DefaultClient,
	}

	ledgers, err := horizon.Ledgers(client.LedgerRequest{
		Order: client.OrderDesc,
		Limit: 1,
	})

	if err != nil {
		panic(err)
	}

	return ledgers.Embedded.Records[0]
}

func getPathWithCursor(path, cursor string) string {
	urlObj, err := url.Parse(path)
	if err != nil {
		panic(err)
	}

	// Add cursor if not present
	q := urlObj.Query()
	if q.Get("cursor") == "" {
		q.Set("cursor", cursor)
	}

	urlObj.RawQuery = q.Encode()
	return urlObj.String()
}

func getPathFromAccessLog(line string) string {
	matches := pathAccessLog.FindStringSubmatch(line)
	if len(matches) != 2 {
		panic("Can't find match: " + line)
	}

	return matches[1]
}

func addPathsFromResponse(a *cmp.Response, level int) {
	newPaths := a.GetPaths()
	for _, newPath := range newPaths {
		// For all indexes with chronological sort ignore order=asc
		// without cursor. There will always be a diff if Horizon started
		// at a different ledger.
		if strings.Contains(newPath, "/ledgers") ||
			strings.Contains(newPath, "/transactions") ||
			strings.Contains(newPath, "/operations") ||
			strings.Contains(newPath, "/payments") ||
			strings.Contains(newPath, "/effects") ||
			strings.Contains(newPath, "/trades") {
			u, err := url.Parse(newPath)
			if err != nil {
				panic(err)
			}

			if u.Query().Get("cursor") == "" &&
				(u.Query().Get("order") == "" || u.Query().Get("order") == "asc") {
				return
			}
		}

		if (strings.Contains(newPath, "/transactions") ||
			strings.Contains(newPath, "/operations") ||
			strings.Contains(newPath, "/payments")) && !strings.Contains(newPath, "include_failed") {
			prefix := "?"
			if strings.Contains(newPath, "?") {
				prefix = "&"
			}

			paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=false", level, false})
			paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=false", level, true})

			paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=true", level, false})
			paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=true", level, true})
			return
		}

		paths = append(paths, pathWithLevel{newPath, level, false})
		paths = append(paths, pathWithLevel{newPath, level, true})
	}
}
