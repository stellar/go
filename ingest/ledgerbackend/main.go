package ledgerbackend

import (
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
)

type isDir interface {
	IsDir() bool
}

type systemCaller interface {
	removeAll(path string) error
	writeFile(filename string, data []byte, perm fs.FileMode) error
	mkdirAll(path string, perm os.FileMode) error
	stat(name string) (isDir, error)
	command(name string, arg ...string) cmdI
}

type realSystemCaller struct{}

func (realSystemCaller) removeAll(path string) error {
	return os.RemoveAll(path)
}

func (realSystemCaller) writeFile(filename string, data []byte, perm fs.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (realSystemCaller) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (realSystemCaller) stat(name string) (isDir, error) {
	return os.Stat(name)
}

func (realSystemCaller) command(name string, arg ...string) cmdI {
	cmd := exec.Command(name, arg...)
	return &realCmd{Cmd: cmd}
}

type cmdI interface {
	Output() ([]byte, error)
	Wait() error
	Start() error
	Run() error
	setDir(dir string)
	setStdout(stdout *logLineWriter)
	getStdout() *logLineWriter
	setStderr(stderr *logLineWriter)
	getStderr() *logLineWriter
	getProcess() *os.Process
	setExtraFiles([]*os.File)
}

type realCmd struct {
	*exec.Cmd
	stdout, stderr *logLineWriter
}

func (r *realCmd) setDir(dir string) {
	r.Cmd.Dir = dir
}

func (r *realCmd) setStdout(stdout *logLineWriter) {
	r.stdout = stdout
	r.Cmd.Stdout = stdout
}

func (r *realCmd) getStdout() *logLineWriter {
	return r.stdout
}

func (r *realCmd) setStderr(stderr *logLineWriter) {
	r.stderr = stderr
	r.Cmd.Stderr = stderr
}

func (r *realCmd) getStderr() *logLineWriter {
	return r.stderr
}

func (r *realCmd) getProcess() *os.Process {
	return r.Cmd.Process
}

func (r *realCmd) setExtraFiles(extraFiles []*os.File) {
	r.ExtraFiles = extraFiles
}
