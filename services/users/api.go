//go:generate gogenv2 server -ext=.server-gen.go -convert_ns=booclient. api.go
//go:generate gogenv2 client -ext=.client-gen.go -convert_ns=booclient. api.go

package users

import (
	"context"
	"net/http"

	"github.com/boo-admin/boo/booclient"
)

type TimeRange = booclient.TimeRange
type Department = booclient.Department
type User = booclient.User
type Role = booclient.Role
type Employee = booclient.Employee
type OperationLog = booclient.OperationLog
type OperationLogLocaleConfig = booclient.OperationLogLocaleConfig
type OperationLogRecord = booclient.OperationLogRecord
type ChangeRecord = booclient.ChangeRecord
type CustomField = booclient.CustomField

type Users interface {
	booclient.Users
	UsersForHTTP
}

type UsersForHTTP interface {
	// @Summary 下载一个用户列表
	// @Param   format             path  string                     false     "下载文件要格式" enums(csv,xlsx)
	// @Param   inline             query bool                       false     "是否作为 body 返回"
	// @Param   sort               query string                       false        "排序字段"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Accept  json
	// @Produce json
	// @Router  /users/export/{format} [get]
	// @x-gogen-noreturn true
	Export(ctx context.Context, format string, inline bool, sort string, offset, limit int64, writer http.ResponseWriter) error

	// @Summary 上传一份用户列表，并创建（或更新）用户信息
	// @Accept  json
	// @Produce json
	// @Router  /users/import [post]
	Import(ctx context.Context, request *http.Request) error
}

type Employees interface {
	booclient.Employees
	EmployeesForHTTP
}

type EmployeesForHTTP interface {
	// @Summary 下载一个员工列表
	// @Param   format             path  string                     false     "下载文件要格式" enums(csv,xlsx)
	// @Param   inline             query bool                       false     "是否作为 body 返回"
	// @Param   sort               query string                       false        "排序字段"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Accept  json
	// @Produce json
	// @Router  /employees/export/{format} [get]
	// @x-gogen-noreturn true
	Export(ctx context.Context, format string, inline bool, sort string, offset, limit int64, writer http.ResponseWriter) error

	// @Summary 上传一份员工列表，并创建（或更新）员工信息
	// @Accept  json
	// @Produce json
	// @Router  /employees/import [post]
	Import(ctx context.Context, request *http.Request) error
}
