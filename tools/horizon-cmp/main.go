package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	client "github.com/stellar/go/clients/horizonclient"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	cmp "github.com/stellar/go/tools/horizon-cmp/internal"
)

// maxLevels defines the maximum number of levels deep the crawler
// should go. Here's an example crawl stack:
// Level 1 = /ledgers?order=desc (finds a link to tx details)
// Level 2 = /transactions/abcdef (finds a link to a list of operations)
// Level 3 = /transactions/abcdef/operations (will not follow any links - at level 3)
const maxLevels = 3
const pathsQueueCap = 10000

// pathAccessLog is a regexp that gets path from ELB access log line. Example:
// 2015-05-13T23:39:43.945958Z my-loadbalancer 192.168.131.39:2817 10.0.0.1:80 0.000086 0.001048 0.001337 200 200 0 57 "GET https://www.example.com:443/transactions?order=desc HTTP/1.1" "curl/7.38.0" DHE-RSA-AES128-SHA TLSv1.2
var pathAccessLog = regexp.MustCompile(`([A-Z]+) http[s]?:\/\/[^/]*(/[^ ]*)`)

var (
	paths                     = make(chan cmp.PathWithLevel, pathsQueueCap)
	visitedPaths              map[string]bool
	elbAccessLogFileReadMutex sync.Mutex
)

// CLI params
var (
	horizonBase           string
	horizonTest           string
	elbAccessLogFile      string
	elbAccessLogstartLine int
	requestsPerSecond     int
)

var rootCmd = &cobra.Command{
	Use:   "horizon-cmp",
	Short: "horizon-cmp compares two horizon servers' responses",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd)
	},
}

func init() {
	visitedPaths = make(map[string]bool)

	rootCmd.Flags().StringVarP(&horizonBase, "base", "b", "", "URL of the base/old version Horizon server")
	rootCmd.Flags().StringVarP(&horizonTest, "test", "t", "", "URL of the test/new version Horizon server")
	rootCmd.Flags().StringVarP(&elbAccessLogFile, "elb-access-log-file", "a", "", "ELB access log file to replay")
	rootCmd.Flags().IntVarP(&elbAccessLogstartLine, "elb-access-start-line", "s", 1, "Start line of ELB access log (useful to continue from a given point)")
	rootCmd.Flags().IntVar(&requestsPerSecond, "rps", 1, "Requests per second")
}

func main() {
	rootCmd.Execute()
}

func run(cmd *cobra.Command) {
	if horizonBase == "" || horizonTest == "" {
		fmt.Print("--base and --test params are required\n\n")
		cmd.Help()
		os.Exit(1)
	}

	// Get latest ledger and operate on it's cursor to get responses at a given ledger.
	ledger := getLatestLedger()
	cursor := ledger.PagingToken()

	var accessLog *cmp.Scanner
	if elbAccessLogFile == "" {
		for _, p := range initPaths {
			paths <- cmp.PathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: false}
			paths <- cmp.PathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: true}
		}
	} else {
		file, err := os.Open(elbAccessLogFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		accessLog = &cmp.Scanner{Scanner: scanner}
		// Seek
		if elbAccessLogstartLine > 1 {
			fmt.Println("Seeking file...")
		}
		for i := 1; i < elbAccessLogstartLine; i++ {
			accessLog.Scan()
		}
		// Streams lines to channel from another go routine
		go streamFile(accessLog)
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

	var wg sync.WaitGroup
	for {
		pl, more := <-paths
		if !more {
			break
		}

		if pl.Level > maxLevels {
			continue
		}

		if visitedPaths[pl.ID()] {
			continue
		}
		visitedPaths[pl.ID()] = true

		time.Sleep(time.Second / time.Duration(requestsPerSecond))
		wg.Add(1)
		go func() {
			defer wg.Done()
			var status strings.Builder
			if accessLog != nil {
				status.WriteString(fmt.Sprintf("%d ", pl.Line))
			}
			status.WriteString(fmt.Sprintf("[stream=%t] %s ", pl.Stream, pl.Path))

			a := cmp.NewResponse(horizonBase, pl.Path, pl.Stream)
			status.WriteString(".")
			b := cmp.NewResponse(horizonTest, pl.Path, pl.Stream)
			status.WriteString(".")

			if a.Equal(b) {
				status.WriteString("ok")
			} else {
				status.WriteString("diff")
				a.SaveDiff(outputDir, b)
			}
			status.WriteString(fmt.Sprintf(" %d %d %d", a.StatusCode, a.Size(), b.Size()))

			fmt.Println(status.String())

			// Add new paths (only for non-ELB)
			if accessLog == nil {
				addPathsFromResponse(a, pl.Level+1)
			}
		}()
	}

	wg.Wait()
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

func getPathFromAccessLog(line string) (string, error) {
	matches := pathAccessLog.FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", errors.Errorf("Can't find match: %s", line)
	}

	if matches[1] != "GET" {
		return "", nil
	}

	return matches[2], nil
}

func streamFile(accessLog *cmp.Scanner) {
	for accessLog.Scan() {
		p := accessLog.Text()
		path, err := getPathFromAccessLog(p)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if path == "" {
			continue
		}

		paths <- cmp.PathWithLevel{Path: path, Level: 0, Line: accessLog.LinesRead(), Stream: false}
		paths <- cmp.PathWithLevel{Path: path, Level: 0, Line: accessLog.LinesRead(), Stream: true}
	}

	if err := accessLog.Err(); err != nil {
		panic("Invalid input: " + err.Error())
	}

	close(paths)
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

			paths <- cmp.PathWithLevel{newPath + prefix + "include_failed=false", level, 0, false}
			paths <- cmp.PathWithLevel{newPath + prefix + "include_failed=false", level, 0, true}

			paths <- cmp.PathWithLevel{newPath + prefix + "include_failed=true", level, 0, false}
			paths <- cmp.PathWithLevel{newPath + prefix + "include_failed=true", level, 0, true}
			return
		}

		paths <- cmp.PathWithLevel{newPath, level, 0, false}
		paths <- cmp.PathWithLevel{newPath, level, 0, true}
	}
}
