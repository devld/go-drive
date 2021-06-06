package utils

import (
	"os"
)

func NewTempFile(file *os.File) *TempFile {
	return &TempFile{file}
}

// TempFile is a marker struct that indicates this file can be moved to another location instead of copying it
type TempFile struct {
	*os.File
}

// TransferTo transfers the file to name by moving.
// If the file is opened, is will be closed before moving
func (t *TempFile) TransferTo(name string) (bool, error) {
	_ = t.File.Close()
	e := os.Rename(t.File.Name(), name)
	if e == nil {
		return true, nil
	}
	open, e := os.Open(t.File.Name())
	if e != nil {
		return false, e
	}
	t.File = open
	return false, nil
}
