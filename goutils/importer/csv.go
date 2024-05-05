package importer

import (
	"encoding/csv"
	"io"

	"github.com/boo-admin/boo/errors"
)

func NewCsvWriter(writer io.Writer) (Writer, error) {
	return csvWriter{out: csv.NewWriter(writer)}, nil
}

type csvWriter struct {
	closer io.Closer
	out    *csv.Writer
}

func (cw csvWriter) Close() error {
	err1 := cw.Flush()
	if cw.closer == nil {
		return err1
	}
	err2 := cw.closer.Close()
	return errors.Join(err1, err2)
}

func (cw csvWriter) Flush() error {
	cw.out.Flush()
	return cw.out.Error()
}

func (cw csvWriter) WriteTitle(record []string) error {
	return cw.out.Write(record)
}

func (cw csvWriter) Write(record []string) error {
	return cw.out.Write(record)
}
