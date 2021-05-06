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
func (t *TempFile) TransferTo(name string) error {
	_ = t.File.Close()
	return os.Rename(t.File.Name(), name)
}
