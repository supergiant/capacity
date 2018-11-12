package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Package specific errors:
var (
	ErrInvalidFilePath = errors.New("invalid filepath")
)

// IsNotExist is a wrapper for os.IsNotExist.
func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// FSFile represents a filesystem file.
type FSFile struct {
	path        string
	permissions os.FileMode
}

// New creates the new FSFile from specified file path and permissions.
func New(path string, fm os.FileMode) (*FSFile, error) {
	if filepath.Base(path) == "." || filepath.Base(path) == string(os.PathSeparator) {
		return nil, errors.Wrap(ErrInvalidFilePath, path)
	}
	return &FSFile{
		path:        path,
		permissions: fm.Perm(),
	}, nil
}

// Info describes a file path.
func (f FSFile) Info() string {
	return fmt.Sprintf("%q file", f.path)
}

// Read reads the file on provided path and returns the contents.
func (f FSFile) Read() ([]byte, error) {
	return ioutil.ReadFile(f.path)
}

// Write stores data to file. If the file does not exist it will be created,
// otherwise it truncates the file before writing.
func (f FSFile) Write(data []byte) error {
	return ioutil.WriteFile(f.path, data, f.permissions)
}
