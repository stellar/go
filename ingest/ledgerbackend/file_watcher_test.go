package ledgerbackend

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stellar/go/support/log"

	"github.com/stretchr/testify/assert"
)

type mockFile struct {
	modTime time.Time
}

func (mockFile) Name() string {
	return ""
}

func (mockFile) Size() int64 {
	return 0
}

func (mockFile) Mode() os.FileMode {
	return 0
}

func (mockFile) IsDir() bool {
	return false
}

func (mockFile) Sys() interface{} {
	return nil
}
func (m mockFile) ModTime() time.Time {
	return m.modTime
}

type mockStat struct {
	sync.Mutex
	t            *testing.T
	expectedPath string
	modTime      time.Time
	err          error
	callCount    int
}

func (m *mockStat) setResponse(modTime time.Time, err error) {
	m.Lock()
	defer m.Unlock()
	m.modTime = modTime
	m.err = err
}

func (m *mockStat) getCallCount() int {
	m.Lock()
	defer m.Unlock()
	return m.callCount
}

func (m *mockStat) stat(fp string) (os.FileInfo, error) {
	m.Lock()
	defer m.Unlock()
	m.callCount++
	assert.Equal(m.t, m.expectedPath, fp)
	//defer m.onCall(m)
	return mockFile{m.modTime}, m.err
}

func createFWFixtures(t *testing.T) (*mockStat, *stellarCoreRunner, *fileWatcher) {
	ms := &mockStat{
		modTime:      time.Now(),
		expectedPath: "/some/path",
		t:            t,
	}

	captiveCoreToml, err := NewCaptiveCoreToml(CaptiveCoreTomlParams{})
	assert.NoError(t, err)

	runner, err := newStellarCoreRunner(CaptiveCoreConfig{
		BinaryPath:         "/some/path",
		HistoryArchiveURLs: []string{"http://localhost"},
		Log:                log.New(),
		Context:            context.Background(),
		Toml:               captiveCoreToml,
	}, stellarCoreRunnerModeOffline)
	assert.NoError(t, err)

	fw, err := newFileWatcherWithOptions(runner, ms.stat, time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 1, ms.getCallCount())

	return ms, runner, fw
}

func TestNewFileWatcherError(t *testing.T) {
	ms := &mockStat{
		modTime:      time.Now(),
		expectedPath: "/some/path",
		t:            t,
	}
	ms.setResponse(time.Time{}, fmt.Errorf("test error"))

	captiveCoreToml, err := NewCaptiveCoreToml(CaptiveCoreTomlParams{})
	assert.NoError(t, err)

	runner, err := newStellarCoreRunner(CaptiveCoreConfig{
		BinaryPath:         "/some/path",
		HistoryArchiveURLs: []string{"http://localhost"},
		Log:                log.New(),
		Context:            context.Background(),
		Toml:               captiveCoreToml,
	}, stellarCoreRunnerModeOffline)
	assert.NoError(t, err)

	_, err = newFileWatcherWithOptions(runner, ms.stat, time.Millisecond)
	assert.EqualError(t, err, "could not stat captive core binary: test error")
	assert.Equal(t, 1, ms.getCallCount())
}

func TestFileChanged(t *testing.T) {
	ms, _, fw := createFWFixtures(t)

	modTime := ms.modTime

	assert.False(t, fw.fileChanged())
	assert.False(t, fw.fileChanged())
	assert.Equal(t, 3, ms.getCallCount())

	ms.setResponse(time.Time{}, fmt.Errorf("test error"))
	assert.False(t, fw.fileChanged())
	assert.Equal(t, 4, ms.getCallCount())

	ms.setResponse(modTime, nil)
	assert.False(t, fw.fileChanged())
	assert.Equal(t, 5, ms.getCallCount())

	ms.setResponse(time.Now().Add(time.Hour), nil)
	assert.True(t, fw.fileChanged())
	assert.Equal(t, 6, ms.getCallCount())
}

func TestCloseRunnerBeforeFileWatcherLoop(t *testing.T) {
	_, runner, fw := createFWFixtures(t)

	assert.NoError(t, runner.close())

	// loop should exit almost immediately because the runner is closed
	fw.loop()
}

func TestCloseRunnerDuringFileWatcherLoop(t *testing.T) {
	ms, runner, fw := createFWFixtures(t)
	done := make(chan struct{})
	go func() {
		fw.loop()
		close(done)
	}()

	// fw.loop will repeatedly check if the file has changed by calling stat.
	// This test ensures that closing the runner will exit fw.loop so that the goroutine is not leaked.

	closedRunner := false
	for {
		select {
		case <-done:
			assert.True(t, closedRunner)
			return
		default:
			if ms.getCallCount() > 20 {
				runner.close()
				closedRunner = true
			}
		}
	}
}

func TestFileChangesTriggerRunnerClose(t *testing.T) {
	ms, runner, fw := createFWFixtures(t)
	done := make(chan struct{})
	go func() {
		fw.loop()
		close(done)
	}()

	// fw.loop will repeatedly check if the file has changed by calling stat.
	// This test ensures that modifying the file will trigger the closing of the runner.
	modifiedFile := false
	for {
		select {
		case <-done:
			assert.True(t, modifiedFile)
			// the runner is closed if and only if runner.ctx.Err() is non-nil
			assert.Error(t, runner.ctx.Err())
			return
		default:
			if ms.getCallCount() > 20 {
				ms.setResponse(time.Now().Add(time.Hour), nil)
				modifiedFile = true
			}
		}
	}
}
