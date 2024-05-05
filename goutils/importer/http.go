package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// MIME types
const (
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMETextXML                          = "text/xml"
	MIMETextXMLCharsetUTF8               = MIMETextXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextCSV                          = "text/csv"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"

	charsetUTF8 = "charset=UTF-8"
)

type contextToRealDir string

func (contextToRealDir) constContextToRealDirKey() {} // nolint:unused

const (
	defaultMemory = 32 << 20 // 32 MB

	ContextToRealDirKey = contextToRealDir("to-real-dir")
)

func ReadHTTP(ctx context.Context, request *http.Request) (Reader, io.Closer, error) {
	queryParams := request.URL.Query()
	ctype := request.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		localfile := false
		switch strings.ToLower(queryParams.Get("localfile")) {
		case "yes", "on", "true", "1":
			localfile = true
		}

		if localfile {
			//  这个是用 el-upload 时的一个场景， el-upload 是选上传， 然后再提效
			// 我们系统有一个自带的 upload server api,  url格式为 /xxxx/api/upload
			var data struct {
				Filename     string `json:"filename"`
				FileEncoding string `json:"file_encoding"`
				Sheet        string `json:"sheet"`
			}
			decoder := json.NewDecoder(request.Body)
			err := decoder.Decode(&data)
			if err != nil {
				return nil, nil, errors.Wrap(err, "读参数失败")
			}

			filename := data.Filename
			if o := ctx.Value(ContextToRealDirKey); o != nil {
				toRealDir := o.(func(context.Context, string) string)
				filename = toRealDir(ctx, filename)
			}
			return ReadFile(ctx, filename, data.FileEncoding, data.Sheet, nil)
		}
	case strings.HasPrefix(ctype, MIMETextCSV):
		var reader io.Reader = request.Body
		if fileEncoding := queryParams.Get("file_encoding"); fileEncoding == "" {
			transformer := unicode.BOMOverride(encoding.Nop.NewDecoder())
			reader = transform.NewReader(request.Body, transformer)
		} else {
			charsetEncoding := charset.Get(fileEncoding)
			if charsetEncoding != nil {
				reader = transform.NewReader(request.Body, charsetEncoding.NewDecoder())
			}
		}
		return ReadCSV(ctx, "body", reader)
	case strings.HasPrefix(ctype, MIMEApplicationForm), strings.HasPrefix(ctype, MIMEMultipartForm):
		err := request.ParseMultipartForm(defaultMemory)
		if err != nil {
			return nil, nil, errors.Wrap(err, "读参数失败")
		}

		form := request.MultipartForm
		if err != nil {
			return nil, nil, errors.Wrap(err, "读参数失败")
		}
		for name, files := range form.File {
			for _, file := range files {
				ext := filepath.Ext(file.Filename)
				switch strings.ToLower(ext) {
				case ".csv":
					fileReader, err := file.Open()
					if err != nil {
						return nil, nil, errors.Wrap(err, "读附件 "+name+" 失败")
					}
					// defer fileReader.Close()

					var reader io.Reader = fileReader
					fileEncoding := queryParams.Get("file_encoding")
					if fileEncoding == "" {
						transformer := unicode.BOMOverride(encoding.Nop.NewDecoder())
						reader = transform.NewReader(fileReader, transformer)
					} else {
						charsetEncoding := charset.Get(fileEncoding)
						if charsetEncoding != nil {
							reader = transform.NewReader(fileReader, charsetEncoding.NewDecoder())
						}
					}
					return ReadCSV(ctx, name, reader)
				case ".xlsx":
					fileReader, err := file.Open()
					if err != nil {
						return nil, nil, errors.Wrap(err, "读附件 "+name+" 失败")
					}
					// defer fileReader.Close()

					sheetName := queryParams.Get("sheet")
					return ReadXlsx(ctx, name, sheetName, fileReader)

				case ".xls":
					fileReader, err := file.Open()
					if err != nil {
						return nil, nil, errors.Wrap(err, "读附件 "+name+" 失败")
					}
					// defer fileReader.Close()

					sheetIdx, _ := strconv.Atoi(queryParams.Get("sheet"))
					fileEncoding := queryParams.Get("file_encoding")
					return ReadXls(ctx, name, fileEncoding, sheetIdx, fileReader)
				default:
					return nil, nil, errors.New("读附件 " + name + " 失败: 文件格式不支持，必须是 xlsx 或 csv 文件")
				}
			}
		}

		return nil, nil, errors.New("读文件失败: 没有附件文件")
	}
	return nil, nil, errors.WithCode(errors.New("Unsupported Media Type - "+ctype), http.StatusUnsupportedMediaType)
}

func ReadFile(ctx context.Context, filename, fileEncoding, sheet string, reader io.Reader) (Reader, io.Closer, error) {
	ext := filepath.Ext(filename)
	switch strings.ToLower(ext) {
	case ".csv":
		if fileEncoding == "" {
			transformer := unicode.BOMOverride(encoding.Nop.NewDecoder())
			reader = transform.NewReader(reader, transformer)
		} else {
			charsetEncoding := charset.Get(fileEncoding)
			if charsetEncoding != nil {
				reader = transform.NewReader(reader, charsetEncoding.NewDecoder())
			}
		}
		return ReadCSV(ctx, filename, reader)
	case ".xlsx":
		return ReadXlsx(ctx, filename, sheet, reader)
	case ".xls":
		sheetIdx, _ := strconv.Atoi("sheet")

		readSeeker, ok := reader.(io.ReadSeeker)
		if !ok {
			return nil, nil, errors.New("读附件 " + filename + " 失败: 参数 reader 必须实现 io.ReadSeeker 接口")
		}
		return ReadXls(ctx, filename, fileEncoding, sheetIdx, readSeeker)
	default:
		return nil, nil, errors.New("读附件 " + filename + " 失败: 文件格式不支持，必须是 xlsx 或 csv 文件")
	}
}

type Recorder interface {
	Open(ctx context.Context) (RecordIterator, []string, error)
}

type RecordIterator interface {
	io.Closer

	Next(ctx context.Context) bool

	Read(ctx context.Context) ([]string, error)
}

type RecorderFunc func(ctx context.Context) (RecordIterator, []string, error)

func (f RecorderFunc) Open(ctx context.Context) (RecordIterator, []string, error) {
	return f(ctx)
}

type RecorderFuncIterator struct {
	CloseFunc func() error
	NextFunc  func(ctx context.Context) bool
	ReadFunc  func(ctx context.Context) ([]string, error)
}

func (s RecorderFuncIterator) Close() error {
	if s.CloseFunc == nil {
		return nil
	}
	return s.CloseFunc()
}
func (s RecorderFuncIterator) Next(ctx context.Context) bool {
	return s.NextFunc(ctx)
}
func (s RecorderFuncIterator) Read(ctx context.Context) ([]string, error) {
	return s.ReadFunc(ctx)
}

func WriteHTTP(ctx context.Context, filename, format string, inline bool, response http.ResponseWriter, recorder Recorder) error {
	var buf = bytes.NewBuffer(make([]byte, 0, 8*1024))
	var out Writer
	var err error
	var contentType string

	switch format {
	case "csv":
		contentType = MIMETextCSV
		out, err = NewCsvWriter(buf)
	case "xlsx":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		out, err = NewXlsxWriter("", buf)
	default:
		return errors.New("'" + format + "' is invalid format")
	}
	if err != nil {
		return err
	}
	defer out.Close()

	iterator, titles, err := recorder.Open(ctx)
	if err != nil {
		return err
	}
	defer iterator.Close()

	// list, err := records.List(ctx, departmentID, 0, 0, state, search, createdAt, 0, 0, "")
	// if err != nil {
	// 	return err
	// }
	// titles := []string{
	// 	"工单类型",
	// 	"部门处室",
	// 	"用户",
	// 	"座机号",
	// 	"申请原因",
	// 	"接单时间",
	// 	"处理结果",
	// 	"处理时间",
	// 	"完成时间",
	// }

	err = out.WriteTitle(titles)
	if err != nil {
		return err
	}

	// departmentCache := map[int64]*XldwDepartment{}
	// userCache := map[int64]*XldwUser{}

	for iterator.Next(ctx) {
		// user := userCache[list[idx].RequesterID]
		// if user == nil {
		// 	u, err := records.users.FindByID(ctx, list[idx].RequesterID)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	userCache[list[idx].RequesterID] = u
		// 	user = u
		// }

		// department := departmentCache[user.DepartmentID]
		// if department == nil {
		// 	d, err := records.departments.FindByID(ctx, user.DepartmentID)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	departmentCache[user.DepartmentID] = d
		// 	department = d
		// }

		// record := []string{
		// 	ToQuestClassLabel(list[idx].QuestClass),
		// 	department.Name,
		// 	user.Name,
		// 	user.Phone,
		// 	list[idx].RequestReason,
		// 	formatTime(list[idx].CreatedAt),
		// 	list[idx].ClosedResult,
		// 	formatTime(list[idx].ClosedAt),
		// 	formatTime(list[idx].RevisitedAt),
		// }

		record, err := iterator.Read(ctx)
		if err != nil {
			return err
		}

		err = out.Write(record)
		if err != nil {
			return err
		}
	}

	err = out.Close()
	if err != nil {
		return err
	}
	response.Header().Set("Content-Type", contentType)
	if inline {
		response.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(filename)+"."+format)
	} else {
		response.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filename)+"."+format)
	}
	response.WriteHeader(http.StatusOK)
	_, err = response.Write(buf.Bytes())
	if err != nil {
		log.Println("write buffer to http response error: ", err)
	}
	return nil
}
