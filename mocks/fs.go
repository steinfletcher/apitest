package mocks

import (
	"io/ioutil"
	"os"
)

type FS struct {
	CapturedCreateName   string
	CapturedCreateFile   string
	CapturedMkdirAllPath string
}

func (m *FS) Create(name string) (*os.File, error) {
	m.CapturedCreateName = name
	file, err := ioutil.TempFile("/tmp", "apitest")
	if err != nil {
		panic(err)
	}
	m.CapturedCreateFile = file.Name()
	return file, nil
}

func (m *FS) MkdirAll(path string, perm os.FileMode) error {
	m.CapturedMkdirAllPath = path
	return nil
}
