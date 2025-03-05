package apitest

import (
	"io/fs"
	"os"
)

//An implementation of fs.FS that wraps your OS's filesystem
type OSFS struct {
}
//Calls os.Open
func (OSFS) Open(name string) (file fs.File, err error) {
	return os.Open(name)
}
