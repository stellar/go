package ledgerbackend

import (
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/stellar/go/support/log"
)

type fileWatcher struct {
	pathToFile  string
	duration    time.Duration
	onChange    func()
	exit        <-chan struct{}
	log         *log.Entry
	stat        func(string) (os.FileInfo, error)
	lastModTime time.Time
}

func newFileWatcher(runner *stellarCoreRunner) (*fileWatcher, error) {
	return newFileWatcherWithOptions(runner, os.Stat, 10*time.Second)
}

func newFileWatcherWithOptions(
	runner *stellarCoreRunner,
	stat func(string) (os.FileInfo, error),
	tickerDuration time.Duration,
) (*fileWatcher, error) {
	info, err := stat(runner.executablePath)
	if err != nil {
		return nil, errors.Wrap(err, "could not stat captive core binary")
	}

	once := &sync.Once{}
	return &fileWatcher{
		pathToFile: runner.executablePath,
		duration:   tickerDuration,
		onChange: func() {
			once.Do(func() {
				runner.log.Warnf("detected new version of captive core binary %s , aborting session.", runner.executablePath)
				if err := runner.close(); err != nil {
					runner.log.Warnf("could not close captive core %v", err)
				}
			})
		},
		exit:        runner.ctx.Done(),
		log:         runner.log,
		stat:        stat,
		lastModTime: info.ModTime(),
	}, nil
}

func (f *fileWatcher) loop() {
	ticker := time.NewTicker(f.duration)

	for {
		select {
		case <-f.exit:
			ticker.Stop()
			return
		case <-ticker.C:
			if f.fileChanged() {
				f.onChange()
			}
		}
	}
}

func (f *fileWatcher) fileChanged() bool {
	info, err := f.stat(f.pathToFile)
	if err != nil {
		f.log.Warnf("could not stat %s: %v", f.pathToFile, err)
		return false
	}

	if modTime := info.ModTime(); !f.lastModTime.Equal(modTime) {
		f.log.Infof(
			"detected update to %s. previous file timestamp was %v current timestamp is %v",
			f.pathToFile,
			f.lastModTime,
			modTime,
		)
		return true
	}
	return false
}
