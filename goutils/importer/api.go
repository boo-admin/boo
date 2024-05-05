package importer

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boo-admin/boo/errors"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Column struct {
	Names    []string
	Required bool
	Set      func(ctx context.Context, lineNumber int, originName, value string) error
}

func StrColumn(names []string, required bool, setStr func(ctx context.Context, lineNumber int, originName, value string) error) Column {
	return Column{
		Names:    names,
		Required: required,
		Set:      setStr,
	}
}

func getname(names []string) string {
	if len(names) > 0 {
		return names[0]
	}
	return "<empty>"
}

func IntColumn(names []string, required bool, setInt func(ctx context.Context, lineNumber int, originName string, value int) error) Column {
	return Column{
		Names:    names,
		Required: required,
		Set: func(ctx context.Context, lineNumber int, originName, value string) error {
			i64, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return errors.Wrap(err, "parse field '"+getname(names)+"' error")
			}
			return setInt(ctx, lineNumber, originName, int(i64))
		},
	}
}

func Int64Column(names []string, required bool, setInt func(ctx context.Context, lineNumber int, originName string, value int64) error) Column {
	return Column{
		Names:    names,
		Required: required,
		Set: func(ctx context.Context, lineNumber int, originName, value string) error {
			i64, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return errors.Wrap(err, "parse field '"+getname(names)+"' error")
			}
			return setInt(ctx, lineNumber, originName, i64)
		},
	}
}

func TimeColumn(names []string, required bool, timeFormats []string, timezone *time.Location, setTime func(ctx context.Context, lineNumber int, originName string, value time.Time) error) Column {
	return Column{
		Names:    names,
		Required: required,
		Set: func(ctx context.Context, lineNumber int, originName, value string) error {
			for _, layout := range timeFormats {
				t, e := time.ParseInLocation(layout, value, timezone)
				if nil == e {
					return setTime(ctx, lineNumber, originName, t)
				}
			}
			return errors.New("time value '" + value + "' is invalid")
		},
	}
}

type Row struct {
	Columns []Column
	Else    func(ctx context.Context, lineNumber int, originName, value string) error
	Commit  func(ctx context.Context) error
}

type Reader interface {
	Read() (record []string, err error)
}

func Import(ctx context.Context, filename string, reader Reader, newRecord func(ctx context.Context, lineNumber int) (Row, error)) error {
	values, err := reader.Read()
	if err != nil {
		return err
	}

	row, err := newRecord(ctx, 0)
	if err != nil {
		return err
	}

	// 初始化 columnIndexs
	columnIndexs := make([]int, len(row.Columns))
	columnNames := make([]string, len(row.Columns))
	for idx := range columnIndexs {
		columnIndexs[idx] = -1
	}

	var valueToColumn = make([]int, len(values))
	for idx := range valueToColumn {
		valueToColumn[idx] = -1
	}

	matchedFieldCount := 0
	search := func(values []string) {
		// 初始化 columnIndexs
		for idx := range row.Columns {
			foundIdx := -1
			for vidx := range values {
				for _, name := range row.Columns[idx].Names {
					if name == values[vidx] {
						foundIdx = vidx
						break
					}
				}
				if foundIdx >= 0 {
					break
				}
			}

			if foundIdx >= 0 {
				columnIndexs[idx] = foundIdx
				columnNames[idx] = values[foundIdx]
				valueToColumn[foundIdx] = idx
				matchedFieldCount++
				continue
			}
		}
	}

	search(values)

	var newValues = make([]string, len(values))
	for vidx := range values {
		bs := FromGB18030([]byte(values[vidx]))
		newValues[vidx] = string(bs)
	}
	// oldMatchedFieldCount := matchedFieldCount
	matchedFieldCount = 0
	search(newValues)
	elseNames := values
	if matchedFieldCount > 0 {
		elseNames = newValues
	}

	var missing []string
	for cidx, idx := range columnIndexs {
		if idx < 0 && row.Columns[cidx].Required {
			missing = append(missing, row.Columns[cidx].Names[0])
		}
	}
	if len(missing) > 0 {
		return ColumnMissingError(missing)
	}
	lineNumber := 1
	for {
		values, err = reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		lineNumber++

		row, err := newRecord(ctx, lineNumber)
		if err != nil {
			return err
		}

		for cidx, idx := range columnIndexs {
			if idx < 0 {
				continue
			}

			if len(values) <= idx {
				continue
			}

			err := row.Columns[cidx].Set(ctx, lineNumber, columnNames[cidx], values[idx])
			if err != nil {
				return WrapError(err, lineNumber, columnNames[cidx])
			}
		}

		if row.Else != nil {
			for idx, colIndex := range valueToColumn {
				if colIndex < 0 {
					err := row.Else(ctx, lineNumber, elseNames[idx], values[idx])
					if err != nil {
						return WrapError(err, lineNumber, elseNames[idx])
					}
				}
			}
		}

		if row.Commit != nil {
			err := row.Commit(ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// func forEach(ctx context.Context, filename string, reader io.Reader,
//   cb func(ctx context.Context, index int, values []string) error {
//     ext := filepath.Ext(filename)
//     switch ext {
//     case ".xlsx":
//       return forEachForXlsx(ctx, filename, reader, cb)
//     case ".xls":
//       return forEachForXls(ctx, filename, reader, cb)
//     case ".csv":
//       return forEachForCSV(ctx, filename, reader, cb)
//     default:
//       return errors.New("file '"+filename+"' format is unsupport")
//     }
// }

func ReadCSV(ctx context.Context, filename string, reader io.Reader) (Reader, io.Closer, error) {
	if reader != nil {
		return csv.NewReader(reader), NoCloser(), nil
	}

	r, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	return csv.NewReader(r), r, nil
}

func NewUploadRequest(urlstr string, params map[string]string, nameField, fileName string, file io.Reader) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	formFile, err := writer.CreateFormFile(nameField, fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(formFile, file)
	if err != nil {
		return nil, err
	}

	for key, val := range params {
		err = writer.WriteField(key, val)
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", urlstr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	return req, nil
}

func UploadFile(httpClient *http.Client, urlstr string, params map[string]string, nameField, fileName string, file io.Reader) (*http.Response, error) {
	request, err := NewUploadRequest(urlstr, params, nameField, fileName, file)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return httpClient.Do(request)
}

func ColumnMissingError(fieldNames []string) error {
	return errors.New("缺少必须的字段 '" + strings.Join(fieldNames, ",") + "'")
}

func WrapError(err error, lineNumber int, fieldName string) error {
	return errors.New("第 " + strconv.FormatInt(int64(lineNumber), 10) + " 行的 '" + fieldName + "' 出错: " + err.Error())
}

func FromGB18030(bs []byte) []byte {
	bb, _, e := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), bs)
	if nil != e {
		return bs
	}
	return bb
}

type CloseFunc func() error

func (f CloseFunc) Close() error {
	if f == nil {
		return nil
	}
	return f()
}

func NoCloser() io.Closer {
	return CloseFunc(nil)
}
