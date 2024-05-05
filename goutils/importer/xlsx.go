package importer

import (
	"context"
	"io"
	"os"
	"strconv"

	"github.com/boo-admin/boo/errors"
	"github.com/xuri/excelize/v2"
)

func ReadXlsx(ctx context.Context, filename string, sheetName string, reader io.Reader) (Reader, io.Closer, error) {
	closer := NoCloser()
	if reader == nil {
		r, err := os.Open(filename)
		if err != nil {
			return nil, nil, err
		}
		reader = r
		closer = r
	}

	file, err := excelize.OpenReader(reader)
	if err != nil {
		closer.Close()
		return nil, nil, err
	}

	if sheetName == "" {
		sheetName = file.GetSheetName(0)
	}

	rows, err := file.Rows(sheetName)
	if err != nil {
		file.Close()
		closer.Close()
		return nil, nil, err
	}

	xr := &xlsxReader{
		file:   file,
		rows:   rows,
		closer: closer,
	}
	return xr, xr, nil
}

type xlsxReader struct {
	file   *excelize.File
	rows   *excelize.Rows
	closer io.Closer
}

func (r *xlsxReader) Close() error {
	err1 := r.rows.Close()
	err2 := r.file.Close()
	err3 := r.closer.Close()
	return errors.Join(err1, err2, err3)
}

func (r *xlsxReader) Read() ([]string, error) {
	if !r.rows.Next() {
		return nil, io.EOF
	}
	return r.rows.Columns()
}

// nolint:unused
type xlsxMultSheetReader struct {
	closer io.Closer
	file   *excelize.File
	sheets []string

	idx         int
	currentRows *excelize.Rows
}

// nolint:unused
func (r *xlsxMultSheetReader) Close() error {
	var err1 error
	if r.currentRows != nil {
		err1 = r.currentRows.Close()
	}
	err2 := r.file.Close()
	err3 := r.closer.Close()
	return errors.Join(err1, err2, err3)
}

// nolint:unused
func (r *xlsxMultSheetReader) Read() ([]string, error) {
	if r.currentRows == nil {
		err := r.nextSheet()
		if err != nil {
			return nil, err
		}
	}

	for !r.currentRows.Next() {
		err := r.nextSheet()
		if err != nil {
			return nil, err
		}
	}
	return r.currentRows.Columns()
}

// nolint:unused
func (r *xlsxMultSheetReader) nextSheet() error {
	if r.idx <= 0 && r.currentRows == nil {
		r.idx = 0
	} else {
		r.idx++
	}

	if r.idx <= len(r.sheets) {
		return io.EOF
	}

	rows, err := r.file.Rows(r.sheets[r.idx])
	if err != nil {
		return err
	}
	r.currentRows = rows
	return nil
}

func NewXlsxWriter(sheet string, out io.Writer) (Writer, error) {
	if sheet == "" {
		sheet = "Sheet1"
	}

	file := excelize.NewFile()
	index, err := file.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	// Set active sheet of the workbook.
	file.SetActiveSheet(index)

	return &xlsxWriter{
		file:  file,
		out:   out,
		sheet: sheet,
	}, nil
}

type xlsxWriter struct {
	closer io.Closer
	file   *excelize.File
	out    io.Writer

	sheet    string
	rowIndex int
}

func (xw *xlsxWriter) Close() error {
	err1 := xw.file.Write(xw.out)
	if xw.closer == nil {
		return err1
	}
	err2 := xw.closer.Close()
	return errors.Join(err1, err2)
}

func (xw *xlsxWriter) Flush() error {
	return nil
}

func (xw *xlsxWriter) WriteTitle(record []string) error {
	return xw.write(0, record)
}

func (xw *xlsxWriter) Write(record []string) error {
	err := xw.write(xw.rowIndex+1, record)
	if err != nil {
		return err
	}

	xw.rowIndex++
	return nil
}

func (xw xlsxWriter) write(index int, record []string) error {
	for idx := range record {
		err := xw.file.SetCellStr(xw.sheet, toAxis(idx)+strconv.Itoa(index+1), record[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	AxisStart = int('A')
	AxisEnd   = int('Z')
)

func toAxis(idx int) string {
	if idx < AxisEnd {
		return string(byte(AxisStart + idx))
	}
	prefix := toAxis(idx / (AxisEnd - AxisStart + 1))
	pos := idx % (AxisEnd - AxisStart + 1)
	return prefix + string(byte(AxisStart+pos))
}
