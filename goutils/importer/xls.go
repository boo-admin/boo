package importer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/boo-admin/boo/errors"
	"github.com/shakinm/xlsReader/xls"
)

var useOle = os.Getenv("importer.xls.windows.use_ole") == "true"

var ReadXlsOnWindows func(ctx context.Context, filename string, charset string, sheetIdx int, reader io.ReadSeeker) (Reader, io.Closer, error)

func ReadXls(ctx context.Context, filename string, charset string, sheetIdx int, reader io.ReadSeeker) (Reader, io.Closer, error) {
	if ReadXlsOnWindows != nil {
		if useOle {
			return ReadXlsOnWindows(ctx, filename, charset, sheetIdx, reader)
		}
	}
	return readXls(ctx, filename, charset, sheetIdx, reader)
}

func readXls(ctx context.Context, filename string, charset string, sheetIdx int, reader io.ReadSeeker) (Reader, io.Closer, error) {
	closer := NoCloser()
	if reader == nil {
		r, err := os.Open(filename)
		if err != nil {
			return nil, nil, err
		}
		reader = r
		closer = r
	}

	file, err := xls.OpenReader(reader)
	if err != nil {
		closer.Close()
		return nil, nil, err
	}

	sheet, err := file.GetSheet(sheetIdx)
	if err != nil {
		closer.Close()
		return nil, nil, err
	}

	if sheet == nil {
		closer.Close()
		return nil, nil, errors.New("sheet '" + strconv.FormatInt(int64(sheetIdx), 10) + "' is missing")
	}

	xr := &xlsReader{
		file:   file,
		sheet:  sheet,
		closer: closer,
	}
	return xr, xr, nil
}

type xlsReader struct {
	file   xls.Workbook
	sheet  *xls.Sheet
	closer io.Closer

	idx int
}

func (r *xlsReader) Close() error {
	return r.closer.Close()
}

func (r *xlsReader) Read() ([]string, error) {
	if r.idx > r.sheet.GetNumberRows() {
		return nil, io.EOF
	}

	row, err := r.sheet.GetRow(r.idx)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to get row `%v`", r.idx))
	}

	r.idx++
	if row == nil {
		return []string{}, nil
	}

	capacity := len(row.GetCols())
	if capacity <= 0 {
		capacity = 1
	}
	values := make([]string, 0, capacity)
	for index := 0; index < len(row.GetCols()); index++ {
		cell, err := row.GetCol(index)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to get cell `%v`", index))
		}
		values = append(values, cell.GetString())
	}
	return values, nil
}
