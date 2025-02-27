package booclient

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/as"
)

var (
	WhiteAddressList = CustomField{
		ID:   "white_address_list",
		Name: "登录IP",
	}
	WelcomeURL = CustomField{
		ID:   "welcome",
		Name: "首页",
	}
	Email = CustomField{
		ID:   "email",
		Name: "邮箱",
	}
	Mobile = CustomField{
		ID:    "mobilephone",
		Name:  "手机",
		Alias: []string{"移动电话", "电话", "手机号", "手机号码"},
	}
	Telephone = CustomField{
		ID:    "telephone",
		Name:  "座机",
		Alias: []string{"固定电话", "座机号", "座机号码"},
	}
	IsSupporter = CustomField{
		ID:   "is_supporter",
		Name: "是否为支持人员",

		Type: "int",
		Values: []EnumerationValue{
			EnumerationValue{
				Label: "主业",
				Value: 0,
			},
			EnumerationValue{
				Label: "支持人员",
				Value: 1,
			},
			EnumerationValue{
				Label: "非支持人员",
				Value: 2,
			},
		},
	}

	DefaultUserFields = []CustomField{
		WhiteAddressList,
		WelcomeURL,
		Mobile,
		Telephone,
		Email,
	}

	DefaultEmployeeFields = []CustomField{
		// WhiteAddressList,
		// WelcomeURL,
		Mobile,
		Telephone,
		Email,
		IsSupporter,
	}
)

type CustomField struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// 主要定义导入时的字段别名
	Alias        []string `json:"alias,omitempty"`
	DefaultValue string   `json:"default,omitempty"`

	Type   string             `json:"type,omitempty"`
	Values []EnumerationValue `json:"values,omitempty"`
}

type EnumerationValue struct {
	Label string      `json:"label,omitempty"`
	Alias []string    `json:"alias,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

func EnumerationValueToString(values []EnumerationValue, value interface{}) string {
	svalue := fmt.Sprint(value)
	if value == nil {
		svalue = ""
	}

	for _, v := range values {
		if v.Value == nil {
			if value == nil {
				return v.Label
			}
			if svalue == "" {
				return v.Label
			}
			continue
		}
		if svalue == fmt.Sprint(v.Value) {
			return v.Label
		}
	}

	return svalue
}

func ParseEnumerationValue(values []EnumerationValue, value string) (interface{}, error) {
	for _, v := range values {
		if value == v.Label {
			return v.Value, nil
		}
	}
	return nil, errors.ErrNotFound
}

func ParseCustomFieldValue(f CustomField, value string) (interface{}, error) {
	if len(f.Values) > 0 {
		return ParseEnumerationValue(f.Values, value)
	}
	switch strings.ToLower(f.Type) {
	case "int", "integer":
		i64, err := strconv.ParseInt(value, 10, 64)
		return i64, err
	case "float":
		f64, err := strconv.ParseFloat(value, 64)
		return f64, err
	case "bool", "boolean":
		b, err := as.Bool(value)
		return b, err
	}

	return value, nil
}
