package cmp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

var findResultMetaXDR = regexp.MustCompile(`"result_meta_xdr": "(.*)",`)

// Horizon 0.16.x vs 0.17.0:
// var succ1 = regexp.MustCompile(`\s*"transaction_successful": true,`)
// var succ2 = regexp.MustCompile(`\s*"successful": true,`)
// var succ3 = regexp.MustCompile(`\s*"transaction_count": [0-9]+,`)
// var succ4 = regexp.MustCompile(`\s*"last_modified_ledger": [0-9]+,`)
// var succ5 = regexp.MustCompile(`\s*"public_key": "G.*",`)

type Response struct {
	Domain string
	Path   string

	Body string
	// NormalizedBody is body without parts that identify a single
	// server (ex. domain) and fields known to be different between
	// instances (ex. `result_meta_xdr`).
	NormalizedBody string
}

func NewResponse(domain, path string) *Response {
	response := &Response{
		Domain: domain,
		Path:   path,
	}

	resp, err := http.Get(domain + path)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		panic(resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	response.Body = string(body)

	normalizedBody := response.Body
	// `result_meta_xdr` can differ between core instances (confirmed this with core team)
	normalizedBody = findResultMetaXDR.ReplaceAllString(normalizedBody, "")
	// Remove Horizon URL from the _links
	normalizedBody = strings.Replace(normalizedBody, domain, "", -1)

	// normalizedBody = succ1.ReplaceAllString(normalizedBody, "")
	// normalizedBody = succ2.ReplaceAllString(normalizedBody, "")
	// normalizedBody = succ3.ReplaceAllString(normalizedBody, "")
	// normalizedBody = succ4.ReplaceAllString(normalizedBody, "")
	// normalizedBody = succ5.ReplaceAllString(normalizedBody, "")

	response.NormalizedBody = normalizedBody
	return response
}

func (r *Response) Equal(other *Response) bool {
	return r.NormalizedBody == other.NormalizedBody
}

func (r *Response) SaveDiff(outputDir string, other *Response) {
	if r.Path != other.Path {
		panic("Paths are different")
	}

	fileName := pathToFileName(r.Path)

	if len(fileName) > 100 {
		fileName = fileName[0:100]
	}

	fileA := fmt.Sprintf("%s/%s.old", outputDir, fileName)
	fileB := fmt.Sprintf("%s/%s.new", outputDir, fileName)
	fileDiff := fmt.Sprintf("%s/%s.diff", outputDir, fileName)

	err := ioutil.WriteFile(fileA, []byte(r.Path+"\n\n"+r.Body), 0744)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fileB, []byte(other.Path+"\n\n"+other.Body), 0744)
	if err != nil {
		panic(err)
	}

	out, err := exec.Command("diff", fileA, fileB).Output()
	if err != nil {
		// Ignore, user will generate diff manually
	}

	if len(out) != 0 {
		err = ioutil.WriteFile(fileDiff, out, 0744)
		if err != nil {
			panic(err)
		}
	}
}

// GetPaths finds all URLs in the response body and returns paths
// (without domain).
func (r *Response) GetPaths() []string {
	// escapedDomain := strings.Replace(r.Domain, `\`, `\\`, -1)
	var linksRegexp = regexp.MustCompile(`"` + r.Domain + `(.*?)["{]`)
	found := linksRegexp.FindAllSubmatch([]byte(r.Body), -1)
	links := make([]string, 0, len(found))

	for _, link := range found {
		l := strings.Replace(string(link[1]), "\\u0026", "&", -1)
		links = append(links, l)
	}

	return links
}

func pathToFileName(path string) string {
	path = strings.Replace(path, "/", "_", -1)
	path = strings.Replace(path, "?", "_", -1)
	path = strings.Replace(path, "&", "_", -1)
	path = strings.Replace(path, "=", "_", -1)
	return path
}
