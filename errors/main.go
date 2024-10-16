package errors

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	xerrors "golang.org/x/exp/errors"
	xfmt "golang.org/x/exp/errors/fmt"
)

// 本库只作了一个很简单的 error 封装，但有少了一个很重要的设计，那就是
// EncodeError 不支持 Is(error) 和 Unwarp()

var (
	ErrSkipped            error = fs.SkipAll
	ErrNotFound                 = sql.ErrNoRows
	ErrUnauthorized             = errors.New("unauthorized")
	ErrCacheInvalid             = errors.New("permission cache is invald")
	ErrTagNotFound              = errors.New("permission tag is not found")
	ErrPermissionNotFound       = errors.New("permission is not found")
	ErrPermissionDeny           = errors.New("permission deny")
	ErrAlreadyClosed            = errors.New("server is closed")
	ErrValueNull                = errors.New("value is null")
	ErrExpectedType             = errors.New("Type unexpected")
	ErrUnimplemented            = errors.New("unimplemented")
	ErrValidationError          = WithErrorCode(errors.New("validation error"), http.StatusBadRequest)
)

type Wrapper = xerrors.Wrapper
type ErrorCoder interface {
	ErrorCode() int
}
type HTTPCoder interface {
	HTTPCode() int
}

type withMessage struct {
	err      error
	noparent bool
	msg      string
}

var _ Wrapper = &withMessage{}

func (w *withMessage) Unwrap() error {
	return w.err
}

func (w *withMessage) Error() string {
	if w.noparent {
		return w.msg
	}
	return w.msg + ": " + w.err.Error()
}

func New(msg string) error {
	return errors.New(msg)
}

func WithText(err error, msg string) error {
	if err == nil {
		panic("WithText: err is null")
	}
	return &withMessage{err: err, noparent: true, msg: msg}
}

func Wrap(err error, msg string) error {
	if err == nil {
		panic("WithText: err is null")
	}
	return &withMessage{err: err, msg: msg}
}

func Unwrap(err error) error {
	return xerrors.Unwrap(err)
}

func Is(err, target error) bool {
	c1 := GetErrorCode(err)
	c2 := GetErrorCode(target)

	if c1 != 0 && c1 == c2 {
		return true
	}
	return xerrors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return xerrors.As(err, target)
}

func Opaque(err error) error {
	return xerrors.Opaque(err)
}

func Join(errs ...error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	return errors.Join(errs...)
}

func ErrorArray(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	return errors.Join(errs...)
}

type withCode struct {
	err  error
	code int
}

var _ Wrapper = &withCode{}

func (w *withCode) HTTPCode() int {
	return ToHttpCode(w.code)
}

func ToHttpCode(code int) int {
	if code < 1000 {
		return code
	}
	return code / 1000
}

func (w *withCode) ErrorCode() int {
	return w.code
}

func (w *withCode) Unwrap() error {
	return w.err
}

func (w *withCode) Error() string {
	return w.err.Error()
}

type withKeyValue struct {
	err   error
	name  string
	value interface{}
}

var _ Wrapper = &withKeyValue{}

func (w *withKeyValue) Unwrap() error {
	return w.err
}

func (w *withKeyValue) Error() string {
	return w.err.Error()
}

func WithCode(err error, code int) *withCode {
	return &withCode{err: err, code: code}
}
func WithErrorCode(err error, code int) *withCode {
	return &withCode{err: err, code: code}
}
func WithHTTPCode(err error, code int) *withCode {
	return &withCode{err: err, code: code}
}

func TryGetErrorCode(target error) (int, bool) {
	if target == nil {
		return 0, false
	}
	wc, ok := target.(ErrorCoder)
	if ok {
		return wc.ErrorCode(), true
	}
	hc, ok := target.(HTTPCoder)
	if ok {
		return hc.HTTPCode(), true
	}
	inner, ok := target.(Wrapper)
	if ok {
		return TryGetErrorCode(inner.Unwrap())
	}
	return 0, false
}

func GetErrorCode(target error, defaultCode ...int) int {
	code, ok := TryGetErrorCode(target)
	if ok {
		if len(defaultCode) > 0 {
			return defaultCode[0]
		}
	}
	return code
}

func TryGetHttpCode(target error) (int, bool) {
	if target == nil {
		return 0, false
	}
	hc, ok := target.(HTTPCoder)
	if ok {
		code := hc.HTTPCode()
		if code != 0 {
			return code, true
		}
	}
	wc, ok := target.(ErrorCoder)
	if ok {
		return wc.ErrorCode(), true
	}
	inner, ok := target.(Wrapper)
	if ok {
		return TryGetHttpCode(inner.Unwrap())
	}
	return 0, false
}

func HttpCodeWith(target error, defaultCode ...int) int {
	code, ok := TryGetHttpCode(target)
	if ok {
		return code
	}
	if len(defaultCode) > 0 {
		return defaultCode[0]
	}
	return http.StatusInternalServerError
}

func GetHttpCode(target error) int {
	code, _ := TryGetHttpCode(target)
	return code
}

func WithKeyValue(err error, name string, value interface{}) error {
	return &withKeyValue{err: err, name: name, value: value}
}

func GetKeyValues(target error) map[string]interface{} {
	return getKeyValues(target, nil)
}

func getKeyValues(target error, values map[string]interface{}) map[string]interface{} {
	for {
		if target == nil {
			return values
		}

		if kv, ok := target.(*withKeyValue); ok {
			if values == nil {
				values = map[string]interface{}{}
			}
			if _, exists := values[kv.name]; !exists {
				values[kv.name] = kv.value
			}
			target = kv.Unwrap()
			continue
		}

		if wc, ok := target.(*withCode); ok {
			if values == nil {
				values = map[string]interface{}{}
			}
			if _, exists := values["code"]; !exists {
				values["code"] = wc.code
			}
			target = wc.Unwrap()
			continue
		}

		switch x := target.(type) {
		case interface{ Unwrap() error }:
			target = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				values = getKeyValues(err, values)
			}
			return values
		default:
			return values
		}
	}
}

type withStack struct {
	err   error
	frame xerrors.Frame
}

var _ Wrapper = &withStack{}

func (w *withStack) Unwrap() error {
	return w.err
}

func (w *withStack) Error() string {
	return w.err.Error()
}

func (w *withStack) Format(s fmt.State, verb rune) {
	xfmt.FormatError(s, verb, w)
}

func (w *withStack) FormatError(pr xerrors.Printer) error { // implements xerrors.Formatter
	pr.Print(w.err.Error())
	if pr.Detail() {
		w.frame.Format(pr)
	}
	return nil
}

func WithStack(err error, skip ...int) error {
	if len(skip) > 0 {
		return &withStack{err: err, frame: xerrors.Caller(skip[0])}
	}
	return &withStack{err: err, frame: xerrors.Caller(0)}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func NewOperationReject(name string) error {
	return Wrap(ErrPermissionDeny, "当前用户是否有 '"+name+"' 权限失败")
}

func NewBadArgument(err error, operation, param string, value ...interface{}) error {
	var msg string
	if len(value) > 0 && value[0] != nil {
		msg = "执行方法 '" + operation + "' 时，参数 '" + param + "' 不正确 - '" + fmt.Sprint(value[0]) + "'"
	} else {
		msg = "执行方法 '" + operation + "' 时，参数 '" + param + "' 不正确"
	}
	return &EncodeError{
		Code:    http.StatusBadRequest,
		Message: msg + ": " + err.Error(),
		Fields:  GetKeyValues(err),
	}
}

type EncodeError struct {
	Code      int                    `json:"code,omitempty"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Fields    map[string]interface{} `json:"data,omitempty"`
	Internals []EncodeError          `json:"internals,omitempty"`
}

func (err *EncodeError) ErrorCode() int {
	return err.Code
}

func (err *EncodeError) HTTPCode() int {
	return ToHttpCode(err.Code)
}

func (err *EncodeError) Error() string {
	return err.Message
}

type ConvertToEncodeError interface {
	ToEncodeError(code ...int)  *EncodeError
}

func ToEncodeError(err error, code ...int) *EncodeError {
	cte, ok := err.(ConvertToEncodeError)
	if ok {
		return cte.ToEncodeError(code...)
	}
	return &EncodeError{
		Code:    GetErrorCode(err, code...),
		Message: err.Error(),
		Fields:  GetKeyValues(err),
	}
}
