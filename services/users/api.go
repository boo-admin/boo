//go:generate gogenv2 server -ext=.server-gen.go -convert_ns=booclient. api.go
//go:generate gogenv2 client -ext=.client-gen.go -convert_ns=booclient. api.go

package users

import (
	"context"
	"net/http"

	"github.com/boo-admin/boo/client"
)

type TimeRange = client.TimeRange
type Department = client.Department
type User = client.User
type Employee = client.Employee
type OperationLog = client.OperationLog
type OperationLogLocaleConfig = client.OperationLogLocaleConfig
type OperationLogRecord = client.OperationLogRecord
type ChangeRecord = client.ChangeRecord
type CustomField = client.CustomField

type Users interface {
	client.Users
	UsersForHTTP
}

type UsersForHTTP interface {
	// @Summary 下载一个用户列表
	// @Param   format            path  string                     false     "下载文件要格式" enums(csv,xlsx)
	// @Param   inline            query bool                       false     "是否作为 body 返回"
	// @Accept  json
	// @Produce json
	// @Router  /users/export/{format} [get]
	Export(ctx context.Context, format string, inline bool, writer http.ResponseWriter) error

	// @Summary 上传一份用户列表，并创建（或更新）用户信息
	// @Accept  json
	// @Produce json
	// @Router  /users/import [post]
	Import(ctx context.Context, request *http.Request) error
}

type Employees interface {
	client.Employees
	EmployeesForHTTP
}

type EmployeesForHTTP interface {
	// @Summary 下载一个员工列表
	// @Param   format            path  string                     false     "下载文件要格式" enums(csv,xlsx)
	// @Param   inline            query bool                       false     "是否作为 body 返回"
	// @Accept  json
	// @Produce json
	// @Router  /employees/export/{format} [get]
	Export(ctx context.Context, format string, inline bool, writer http.ResponseWriter) error

	// @Summary 上传一份员工列表，并创建（或更新）员工信息
	// @Accept  json
	// @Produce json
	// @Router  /employees/import [post]
	Import(ctx context.Context, request *http.Request) error
}
