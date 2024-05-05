package importer

import (
	"io"
)

type Writer interface {
	io.Closer

	WriteTitle([]string) error
	Write([]string) error

	Flush() error
}
