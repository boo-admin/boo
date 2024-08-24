// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/get"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	entran "github.com/go-playground/validator/v10/translations/en"
	zhtran "github.com/go-playground/validator/v10/translations/zh"
	"golang.org/x/exp/slog"
	// "github.com/go-playground/validator/v10/translations/fr"
	// "github.com/go-playground/validator/v10/translations/id"
	// "github.com/go-playground/validator/v10/translations/ja"
	// "github.com/go-playground/validator/v10/translations/nl"
	// "github.com/go-playground/validator/v10/translations/pt_BR"
	// "github.com/go-playground/validator/v10/translations/ru"
	// "github.com/go-playground/validator/v10/translations/tr"
	// "github.com/go-playground/validator/v10/translations/zh_tw"
)

var (
	En   ut.Translator
	Zh   ut.Translator
	UtZh *ut.UniversalTranslator
)

type StructValidator = validator.Validate

var DefaultStructValidator = validator.New()

func init() {
	zhLocale := zh.New()
	enLocale := en.New()
	UtZh = ut.New(zhLocale, zhLocale, enLocale)

	Zh, _ = UtZh.GetTranslator("zh")
	En, _ = UtZh.GetTranslator("en")

	err := zhtran.RegisterDefaultTranslations(DefaultStructValidator, Zh)
	if err != nil {
		slog.Error("初始化 zh 本地化校验失败", slog.Any("error", err))
	}
	err = entran.RegisterDefaultTranslations(DefaultStructValidator, En)
	if err != nil {
		slog.Error("初始化 en 本地化校验失败", slog.Any("error", err))
	}
}

type ValidatableError interface {
	IsValidationErrors() bool
	ToValidationErrors() []ValidationError
}

func ToValidationErrors(err error) (bool, []ValidationError) {
	if encodeErr, ok := err.(*errors.EncodeError); ok {
		if encodeErr.Code != errors.ErrValidationError.ErrorCode() {
			return false, nil
		}

		if len(encodeErr.Internals) > 0 {
			var validationErrors = make([]ValidationError, len(encodeErr.Internals))
			for idx := range encodeErr.Internals {
				validationErrors[idx] = *toValidationError(&encodeErr.Internals[idx])
			}
			return true, validationErrors
		}
		return true, []ValidationError{*toValidationError(encodeErr)}
	}
	e, ok := err.(interface {
		ToValidationErrors() []ValidationError
	})
	if ok {
		is, ok := err.(interface {
			IsValidationErrors() bool
		})
		if ok {
			if !is.IsValidationErrors() {
				return false, nil
			}
		}
		return true, e.ToValidationErrors()
	}
	return false, nil
}

type ValidationErrors []ValidationError

func (err ValidationErrors) ToEncodeError(code ...int)  *errors.EncodeError {
	if len(err) == 0 {
		return nil
	}

	var internals = make([]errors.EncodeError, len(err))
	for idx := range internals {
		internals[idx] = *err[idx].ToEncodeError()
	}
	return &errors.EncodeError{
		Code:    errors.ErrValidationError.ErrorCode(),
		Message: "表单参数验证错误",
		Internals:  internals,
	}	
}

func (err ValidationErrors) ErrorCode() int {
	return errors.GetErrorCode(errors.ErrValidationError)
}

func (err ValidationErrors) HTTPCode() int {
	return errors.GetHttpCode(errors.ErrValidationError)
}

func (err ValidationErrors) Is(e error) bool {
	return e == errors.ErrValidationError
}

func (err ValidationErrors) Unwrap() []error {
	var errs = make([]error, len(err))
	for idx := range err {
		errs[idx] = &err[idx]
	}
	return errs
}

func (err ValidationErrors) Error() string {
	var sb strings.Builder
	for idx := range err {
		sb.WriteString(err[idx].Key)
		sb.WriteString(": ")
		sb.WriteString(err[idx].Message)
		sb.WriteString(";")
	}
	return sb.String()
}

func (e *ValidationErrors) IsValidationErrors() bool {
	return true
}

func (err ValidationErrors) ToValidationErrors() []ValidationError {
	return err
}

// ValidationError simple struct to store the Message & Key of a validation error
type ValidationError struct {
	Code, Message, Key string
}

func toValidationError(encodeErr *errors.EncodeError) *ValidationError {
	return &ValidationError{
		Code: get.StringWithDefault(encodeErr.Fields, "validation.code", ""),
		Message: encodeErr.Message,
		Key: get.StringWithDefault(encodeErr.Fields, "validation.key", ""),
	}
}

func (e *ValidationError) ToEncodeError(code ...int)  *errors.EncodeError {
	return &errors.EncodeError{
		Code:    errors.ErrValidationError.ErrorCode(),
		Message: e.Message,
		Fields:  map[string]interface{}{
			"validation.code": e.Code,
			"validation.key": e.Key,
		},
	}	
}

func (e *ValidationError) ErrorCode() int {
	return errors.GetErrorCode(errors.ErrValidationError)
}

func (e *ValidationError) HTTPCode() int {
	return errors.GetHttpCode(errors.ErrValidationError)
}

func (err *ValidationError) Is(e error) bool {
	return e == errors.ErrValidationError
}

// String returns the Message field of the ValidationError struct.
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (err *ValidationError) Unwrap() error {
	return errors.ErrValidationError
}

func (e *ValidationError) IsValidationErrors() bool {
	return true
}

func (e *ValidationError) ToValidationErrors() []ValidationError {
	return []ValidationError{*e}
}

var _ errors.ConvertToEncodeError = &ValidationError{}

func NewValidationError(field, message string) error {
	return &ValidationError{Key: field, Message: message}
}

type TranslateFunc func(locale, message string, args ...interface{}) string

var Default = New("zh", nil)

func New(locale string, translator TranslateFunc) *Validation {
	return &Validation{
		Locale:     locale,
		Translator: translator,
	}
}

// Validation context manages data validation and error messages.
type Validation struct {
	Validator  *StructValidator
	Locale     string
	Translator TranslateFunc
	Errors     []ValidationError
}

// New a new Validation
func (v *Validation) New() *Validation {
	n := &Validation{}
	*n = *v
	n.Clear()
	return n
}

// Clear *all* ValidationErrors
func (v *Validation) Clear() {
	v.Errors = []ValidationError{}
}

// HasErrors returns true if there are any (ie > 0) errors. False otherwise.
func (v *Validation) HasErrors() bool {
	return len(v.Errors) > 0
}

// ErrorMap returns the errors mapped by key.
// If there are multiple validation errors associated with a single key, the
// first one "wins".  (Typically the first validation will be the more basic).
func (v *Validation) ErrorMap() map[string]ValidationError {
	m := map[string]ValidationError{}
	for _, e := range v.Errors {
		if _, ok := m[e.Key]; !ok {
			m[e.Key] = e
		}
	}
	return m
}

func (v *Validation) ToError() error {
	if len(v.Errors) == 0 {
		panic(errors.New("validation errors empty"))
	}
	return ValidationErrors(v.Errors)
}

// Error adds an error to the validation context.
func (v *Validation) Error(field, message string, args ...interface{}) *ValidationResult {
	result := v.validationResult(false)
	v.Errors = append(v.Errors, ValidationError{})
	result.Error = &v.Errors[len(v.Errors)-1]

	result.Message(message, args...)
	result.Error.Key = field
	return result
}

// Error adds an error to the validation context.
func (v *Validation) validationResult(ok bool) *ValidationResult {
	return &ValidationResult{Ok: ok, Locale: v.Locale, Translator: v.Translator}
}

// ValidationResult is returned from every validation method.
// It provides an indication of success, and a pointer to the Error (if any).
type ValidationResult struct {
	Error      *ValidationError
	Ok         bool
	Locale     string
	Translator func(locale, message string, args ...interface{}) string
}

// Key sets the ValidationResult's Error "key" and returns itself for chaining
func (r *ValidationResult) Key(key string) *ValidationResult {
	if r.Error != nil {
		r.Error.Key = key
	}
	return r
}

// Allow a message key to be passed into the validation result. The Validation has already
// setup the translator to translate the message key
func (r *ValidationResult) Message(message string, args ...interface{}) *ValidationResult {
	if r.Error == nil {
		return r
	}

	// If translator found, use that to create the message, otherwise call Message method
	if r.Translator != nil {
		r.Error.Message = r.Translator(r.Locale, message, args...)
	} else if len(args) == 0 {
		r.Error.Message = message
	} else {
		r.Error.Message = fmt.Sprintf(message, args...)
	}
	return r
}

// Required tests that the argument is non-nil and non-empty (if string or list)
func (v *Validation) Required(field string, obj interface{}) *ValidationResult {
	return v.apply(Required{}, field, obj)
}

func (v *Validation) Min(field string, n int, min int) *ValidationResult {
	return v.MinFloat(field, float64(n), float64(min))
}

func (v *Validation) MinFloat(field string, n float64, min float64) *ValidationResult {
	return v.apply(Min{min}, field, n)
}

func (v *Validation) Max(field string, n int, max int) *ValidationResult {
	return v.MaxFloat(field, float64(n), float64(max))
}

func (v *Validation) MaxFloat(field string, n float64, max float64) *ValidationResult {
	return v.apply(Max{max}, field, n)
}

func (v *Validation) Range(field string, n, min, max int) *ValidationResult {
	return v.RangeFloat(field, float64(n), float64(min), float64(max))
}

func (v *Validation) Range64(field string, n, min, max int64) *ValidationResult {
	return v.RangeFloat(field, float64(n), float64(min), float64(max))
}

func (v *Validation) RangeFloat(field string, n, min, max float64) *ValidationResult {
	return v.apply(Range{Min{min}, Max{max}}, field, n)
}

func (v *Validation) TimeStartEndCheck(field string, startTime, endTime time.Time) *ValidationResult {
	return v.apply(TimeStartEndCheck{start: startTime, end: endTime}, field, startTime)
}

func (v *Validation) MinSize(field string, obj interface{}, min int) *ValidationResult {
	return v.apply(MinSize{min}, field, obj)
}

func (v *Validation) MaxSize(field string, obj interface{}, max int) *ValidationResult {
	return v.apply(MaxSize{max}, field, obj)
}

func (v *Validation) Length(field string, obj interface{}, n int) *ValidationResult {
	return v.apply(Length{n}, field, obj)
}

func (v *Validation) Match(field, str string, regex *regexp.Regexp) *ValidationResult {
	return v.apply(Match{regex}, field, str)
}

func (v *Validation) Email(field, str string) *ValidationResult {
	return v.apply(Email{Match{emailPattern}}, field, str)
}

func (v *Validation) IPAddr(field, str string, cktype ...int) *ValidationResult {
	return v.apply(IPAddr{cktype}, field, str)
}

func (v *Validation) MacAddr(field, str string) *ValidationResult {
	return v.apply(IPAddr{}, field, str)
}

func (v *Validation) Domain(field, str string) *ValidationResult {
	return v.apply(Domain{}, field, str)
}

func (v *Validation) URL(field, str string) *ValidationResult {
	return v.apply(URL{}, field, str)
}

func (v *Validation) PureText(field, str string, m int) *ValidationResult {
	return v.apply(PureText{m}, field, str)
}

func (v *Validation) FilePath(field, str string, m int) *ValidationResult {
	return v.apply(FilePath{m}, field, str)
}

func (v *Validation) apply(chk Validator, field string, obj interface{}) *ValidationResult {
	if chk.IsSatisfied(obj) {
		return v.validationResult(true)
	}

	// Also return it in the result.
	result := v.validationResult(false)

	var messageText string
	// If translator found, use that to create the message, otherwise call Message method
	if v.Translator != nil {
		message, args := chk.Message()
		messageText = v.Translator(v.Locale, message, args...)
		if strings.HasPrefix(messageText, "???") {
			messageText = chk.DefaultMessage()
		}
	} else {
		messageText = chk.DefaultMessage()
	}

	// Add the error to the validation context.
	v.Errors = append(v.Errors, ValidationError{
		Message: messageText,
		Key:     field,
	})
	result.Error = &v.Errors[len(v.Errors)-1]
	return result
}

// Check applies a group of validators to a field, in order, and return the
// ValidationResult from the first one that fails, or the last one that
// succeeds.
func (v *Validation) Check(field string, obj interface{}, checks ...Validator) *ValidationResult {
	var result *ValidationResult
	for _, check := range checks {
		result = v.apply(check, field, obj)
		if !result.Ok {
			return result
		}
	}
	return result
}

func (v *Validation) Struct(value interface{}) *Validation {
	var err error
	if v.Validator != nil {
		err = v.Validator.Struct(value)
	} else {
		err = DefaultStructValidator.Struct(value)
	}
	if err == nil {
		return v
	}

	translator, _ := UtZh.GetTranslator(v.Locale)
	messages := err.(validator.ValidationErrors).Translate(translator)

	for key, message := range messages {
		v.Errors = append(v.Errors, ValidationError{
			Key:     key,
			Message: message,
		})
	}
	return v
}
