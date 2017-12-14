package main

// See README.md for a description of this script

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"time"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

var extractBinName = regexp.MustCompile(`^(?P<bin>[a-z-]+)-(?P<tag>.+)$`)

var builds = []buildConfig{
	{"darwin", "amd64"},
	{"linux", "amd64"},
	{"linux", "arm"},
	{"windows", "amd64"},
}

var binFilter = flag.String("bin", "", "restrict build to single binary")
var osFilter = flag.String("os", "", "restrict build to single os")
var archFilter = flag.String("arch", "", "restrict build to single arch")
var keepDir = flag.Bool("keep", false, "when true, artifact directories are not removed after packaging")

type buildConfig struct {
	OS   string
	Arch string
}

func main() {
	flag.Parse()
	log.SetLevel(log.InfoLevel)
	run("rm", "-rf", "dist/*")

	if os.Getenv("TRAVIS_EVENT_TYPE") == "cron" {
		buildNightlies()
		os.Exit(0)
	} else if os.Getenv("TRAVIS_TAG") != "" {
		buildByTag()
		os.Exit(0)
	} else {
		buildSnapshots()
		os.Exit(0)
	}

	log.Info("nothing to do")
}

// package searches the `tools` and `services` packages of this repo to find
// the source directory.  This is used within the script to find the README and
// other files that should be packaged with the binary.
func binPkgNames() []string {
	result := []string{}
	result = append(result, binNamesForDir("services")...)
	result = append(result, binNamesForDir("tools")...)
	return result
}

func binNamesForDir(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(errors.Wrap(err, "read-dir failed"))
	}

	result := []string{}
	for _, file := range files {
		if file.IsDir() {
			result = append(result, filepath.Join(dir, file.Name()))
		}
	}

	return result
}

func build(pkg, dest, version, buildOS, buildArch string) {
	buildTime := time.Now().Format(time.RFC3339)

	timeFlag := fmt.Sprintf("-X github.com/stellar/go/support/app.buildTime=%s", buildTime)
	versionFlag := fmt.Sprintf("-X github.com/stellar/go/support/app.version=%s", version)

	if buildOS == "windows" {
		dest = dest + ".exe"
	}

	cmd := exec.Command("go", "build",
		"-o", dest,
		"-ldflags", fmt.Sprintf("%s %s", timeFlag, versionFlag),
		pkg,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("GOOS=%s", buildOS),
		fmt.Sprintf("GOARCH=%s", buildArch),
	)
	log.Infof("building %s", pkg)

	log.Infof("running: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func buildNightlies() {
	version := runOutput("git", "describe", "--always", "--dirty", "--tags")
	repo := repoName()

	for _, pkg := range binPkgNames() {
		bin := filepath.Base(pkg)

		if *binFilter != "" && *binFilter != bin {
			continue
		}

		for _, cfg := range getBuildConfigs() {
			dest := prepareDest(pkg, bin, "nightly", cfg.OS, cfg.Arch)

			build(
				fmt.Sprintf("%s/%s", repo, pkg),
				filepath.Join(dest, bin),
				version,
				cfg.OS,
				cfg.Arch,
			)

			packageArchive(dest, cfg.OS)
		}
	}
}

func buildByTag() {
	bin, version := extractFromTag(os.Getenv("TRAVIS_TAG"))
	pkg := packageName(bin)
	repo := repoName()

	if bin == "" {
		log.Info("could not extract info from TRAVIS_TAG: skipping artifact packaging")
		os.Exit(0)
	}

	for _, cfg := range getBuildConfigs() {
		dest := prepareDest(pkg, bin, version, cfg.OS, cfg.Arch)

		// rebuild the binary with the version variable set
		build(
			fmt.Sprintf("%s/%s", repo, pkg),
			filepath.Join(dest, bin),
			version,
			cfg.OS,
			cfg.Arch,
		)

		packageArchive(dest, cfg.OS)
	}
}

func buildSnapshots() {
	rev := runOutput("git", "describe", "--always", "--dirty")
	version := fmt.Sprintf("snapshot-%s", rev)
	repo := repoName()

	for _, pkg := range binPkgNames() {
		bin := filepath.Base(pkg)

		if *binFilter != "" && *binFilter != bin {
			continue
		}

		for _, cfg := range getBuildConfigs() {

			//WARNING - DIRTY FILTHY HACK
			//hardcode non-linux bifrost out of the snapshots build
			//TODO: find alternative
			if pkg=="services/bifrost" && !(cfg.OS=="linux" && cfg.Arch=="amd64"){
				log.Info("ignoring biforst for non linux/amd64")
				continue
			}

			dest := prepareDest(pkg, bin, "snapshot", cfg.OS, cfg.Arch)

			build(
				fmt.Sprintf("%s/%s", repo, pkg),
				filepath.Join(dest, bin),
				version,
				cfg.OS,
				cfg.Arch,
			)

			packageArchive(dest, cfg.OS)
		}
	}
}

// extractFromTag extracts the name of the binary that should be packaged in the
// course of execution this script as well as the version it should be packaged
// as, based on the name of the tag in the TRAVIS_TAG environment variable.
// Tags must be of the form `NAME-vSEMVER`, such as `horizon-v1.0.0` to be
// matched by this function.
//
// In the event that the TRAVIS_TAG is missing or the match fails, an empty
// string will be returned.
func extractFromTag(tag string) (string, string) {
	match := extractBinName.FindStringSubmatch(tag)
	if match == nil {
		return "", ""
	}

	return match[1], match[2]
}

func getBuildConfigs() (result []buildConfig) {
	for _, cfg := range builds {

		if *osFilter != "" && *osFilter != cfg.OS {
			continue
		}

		if *archFilter != "" && *archFilter != cfg.Arch {
			continue
		}

		result = append(result, cfg)
	}
	return
}

// packageArchive tars or zips `dest`, depending upon the OS, then removes
// `dest`, in preparation of travis uploading all artifacts to github releases.
func packageArchive(dest, buildOS string) {
	release := filepath.Base(dest)
	dir := filepath.Dir(dest)

	if buildOS == "windows" {
		pop := pushdir(dir)
		// zip $RELEASE.zip $RELEASE/*
		run("zip", "-r", release+".zip", release)
		pop()
	} else {
		// tar -czf $dest.tar.gz -C $DIST $RELEASE
		run("tar", "-czf", dest+".tar.gz", "-C", dir, release)
	}

	if !*keepDir {
		run("rm", "-rf", dest)
	}
}

// package searches the `tools` and `services` packages of this repo to find
// the source directory.  This is used within the script to find the README and
// other files that should be packaged with the binary.
func packageName(binName string) string {
	targets := []string{
		filepath.Join("services", binName),
		filepath.Join("tools", binName),
	}

	var result string

	// Note: we do not short circuit this search when we find a valid result so
	// that we can panic when multiple results are found.  The children of
	// /services and /tools should not have name overlap.
	for _, t := range targets {
		_, err := os.Stat(t)

		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			panic(errors.Wrap(err, "stat failed"))
		}

		if result != "" {
			panic("sourceDir() found multiple results!")
		}

		result = t
	}

	return result
}

func prepareDest(pkg, bin, version, os, arch string) string {
	name := fmt.Sprintf("%s-%s-%s-%s", bin, version, os, arch)
	dest := filepath.Join("dist", name)

	// make destination directories
	run("mkdir", "-p", dest)
	run("cp", "LICENSE-APACHE.txt", dest)
	run("cp", "COPYING", dest)
	run("cp", filepath.Join(pkg, "README.md"), dest)
	run("cp", filepath.Join(pkg, "CHANGELOG.md"), dest)
	return dest
}

// pushdir is a utility function to temporarily change directories.  It returns
// a func that can be called to restore the current working directory to the
// state it was in when first calling pushdir.
func pushdir(dir string) func() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(errors.Wrap(err, "getwd failed"))
	}

	err = os.Chdir(dir)
	if err != nil {
		panic(errors.Wrap(err, "chdir failed"))
	}

	return func() {
		err := os.Chdir(cwd)
		if err != nil {
			panic(errors.Wrap(err, "revert dir failed"))
		}
	}
}

func repoName() string {
	if os.Getenv("REPO") != "" {
		return os.Getenv("REPO")
	}
	return "github.com/stellar/go"

}

// utility command to run the provided command that echoes any output.  A failed
// command will trigger a panic.
func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	log.Infof("running: %s %s", name, strings.Join(args, " "))
	err := cmd.Run()

	if err != nil {
		panic(err)
	}
}

// utility command to run  the provided command that returns the output.  A
// failed command will trigger a panic.
func runOutput(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr

	log.Infof("running: %s %s", name, strings.Join(args, " "))
	out, err := cmd.Output()

	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(out))
}
