package booclient

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/as"
	"github.com/hjson/hjson-go/v4"
	"github.com/runner-mei/resty"
)

type TimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

func (tr TimeRange) IsZero() bool {
	return tr.Start.IsZero() && tr.End.IsZero()
}

func BoolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func ToBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "on"
}

func ToInt64Array(array []string) ([]int64, error) {
	var int64Array []int64
	for _, s := range array {
		ss := strings.Split(s, ",")
		for _, v := range ss {
			if v == "" {
				continue
			}
			i64, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			int64Array = append(int64Array, i64)
		}
	}
	return int64Array, nil
}

func ToBoolArray(array []string) ([]bool, error) {
	var boolArray []bool
	for _, s := range array {
		ss := strings.Split(s, ",")
		for _, v := range ss {
			if v == "" {
				continue
			}
			i64, err := as.Bool(v)
			if err != nil {
				return nil, err
			}
			boolArray = append(boolArray, i64)
		}
	}
	return boolArray, nil
}

func ToDatetime(s string) (time.Time, error) {
	return as.StrToTime(s)
}

func ToDatetimes(array []string) ([]time.Time, error) {
	var timeArray []time.Time
	for _, s := range array {
		ss := strings.Split(s, ",")
		for _, v := range ss {
			if v == "" {
				continue
			}
			t, err := ToDatetime(v)
			if err != nil {
				return nil, err
			}
			timeArray = append(timeArray, t)
		}
	}
	return timeArray, nil
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

func FromHjsonFile(filename string, target interface{}, contentFuncs ...func([]byte) []byte) error {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if bytes.HasPrefix(bs, []byte{0xEF, 0xBB, 0xBF}) {
		// REMOVE bom
		bs = bs[3:]
	}

	if len(contentFuncs) > 0 {
		for _, cb := range contentFuncs {
			bs = cb(bs)
		}
	}

	return hjson.UnmarshalWithOptions(bs, target, hjson.DecoderOptions{
		UseJSONNumber:         true,
		DisallowUnknownFields: false,
		DisallowDuplicateKeys: false,
		WhitespaceAsComments:  true,
	})
}

// FileExists 文件是否存在
func FileExists(dir string, e ...*error) bool {
	info, err := os.Stat(dir)
	if err != nil {
		if len(e) != 0 && e[0] != nil {
			*e[0] = err
		}
		return false
	}

	return !info.IsDir()
}

// DirExists 目录是否存在
func DirExists(dir string, err ...*error) bool {
	d, e := os.Stat(dir)
	switch {
	case e != nil:
		if len(err) != 0 && err[0] != nil {
			*err[0] = e
		}
		return false
	case !d.IsDir():
		return false
	}

	return true
}

func IsDirectory(dir string, e ...*error) bool {
	info, err := os.Stat(dir)
	if err != nil {
		if len(e) != 0 && e[0] != nil {
			*e[0] = err
		}
		return false
	}

	return info.IsDir()
}

func ToResponseError(response *http.Response, msg string) error {
	var sb strings.Builder

	sb.WriteString(msg)
	sb.WriteString(", ")
	sb.WriteString(response.Status)
	sb.WriteString(": ")
	io.Copy(&sb, response.Body)
	return errors.WithCode(errors.New(sb.String()), response.StatusCode)
}

func GetDefaultClient() *http.Client {
	return resty.InsecureHttpClent
}
