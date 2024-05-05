package importer

import (
	"context"
	"io"
	"os"
	"strconv"

	"github.com/boo-admin/boo/errors"
	"github.com/extrame/xls"
)

func init() {
	ReadXlsOnWindows = readXlsWithOle
}

func readXlsWithOle(ctx context.Context, filename string, charset string, sheetIdx int, reader io.ReadSeeker) (Reader, io.Closer, error) {
	closer := NoCloser()
	if reader == nil {
		r, err := os.Open(filename)
		if err != nil {
			return nil, nil, err
		}
		reader = r
		closer = r
	}

	file, err := xls.OpenReader(reader, charset)
	if err != nil {
		closer.Close()
		return nil, nil, err
	}

	sheet := file.GetSheet(sheetIdx)
	if sheet == nil {
		closer.Close()
		return nil, nil, errors.New("sheet '" + strconv.FormatInt(int64(sheetIdx), 10) + "' is missing")
	}

	xr := &xlsReaderOle{
		file:   file,
		sheet:  sheet,
		closer: closer,
	}
	return xr, xr, nil
}

type xlsReaderOle struct {
	file   *xls.WorkBook
	sheet  *xls.WorkSheet
	closer io.Closer

	idx int
}

func (r *xlsReaderOle) Close() error {
	return r.closer.Close()
}

func (r *xlsReaderOle) Read() ([]string, error) {
	if uint16(r.idx) > r.sheet.MaxRow {
		return nil, io.EOF
	}

	row := r.sheet.Row(r.idx)
	r.idx++
	if row == nil {
		return []string{}, nil
	}

	capacity := row.LastCol() - row.FirstCol()
	if capacity <= 0 {
		capacity = 1
	}
	values := make([]string, 0, capacity)
	for index := row.FirstCol(); index < row.LastCol(); index++ {
		values = append(values, row.Col(index))
	}
	return values, nil
}
