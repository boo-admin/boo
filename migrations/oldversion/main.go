package oldversion

import (
	"io/fs"

	rice "github.com/GeertJohan/go.rice"
)

func GetStaticDir() (fs.FS, error) {
	staticFS, err := rice.FindBox("../../migrations")
	if err != nil {
		return nil, err
	}
	return riceFS{staticFS}, nil
}

type riceFS struct {
	*rice.Box
}

func (rfs riceFS) Open(name string) (fs.File, error) {
	return rfs.Box.Open(name)
}
