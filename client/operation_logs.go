//go:generate gogenv2 server -ext=.server-gen.go operation_logs.go
//go:generate gogenv2 client -ext=.client-gen.go operation_logs.go

package client

import (
	"context"
	"database/sql"
	"time"
)

type OperationLog struct {
	TableName  struct{}            `json:"-" xorm:"boo_operation_logs"`
	ID         int64               `json:"id,omitempty" xorm:"id pk autoincr"`
	UserID     int64               `json:"userid,omitempty" xorm:"userid null"`
	Username   string              `json:"username,omitempty" xorm:"username null"`
	Successful bool                `json:"successful" xorm:"successful notnull"`
	Type       string              `json:"type" xorm:"type notnull"`
	TypeTitle  string              `json:"type_title" xorm:"-"`
	Content    string              `json:"content,omitempty" xorm:"content null"`
	Fields     *OperationLogRecord `json:"fields,omitempty" xorm:"fields json null"`
	CreatedAt  time.Time           `json:"created_at,omitempty" xorm:"created_at"`
}

type OperationLogRecord struct {
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   int64          `json:"object_id,omitempty"`
	Records    []ChangeRecord `json:"records,omitempty"`
}

type ChangeRecord struct {
	Name     string      `json:"name"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`

	DisplayName     string      `json:"display_name,omitempty"`
	OldDisplayValue interface{} `json:"old_display_value,omitempty"`
	NewDisplayValue interface{} `json:"new_display_value,omitempty"`
}

type OperationLogLocaleConfig struct {
	Title  string
	Fields map[string]string
}

type OperationQueryer interface {
	// @Summary 操作日志字段本地化信息(此功能暂时没有想好，后面可能会改)
	// @Description 操作日志中各个字段本地化信息
	// @Accept  json
	// @Produce  json
	// @Router /oplog/locales [get]
	// @Success 200 {object} map[string]OperationLogLocaleConfig
	GetLocales(ctx context.Context) (map[string]OperationLogLocaleConfig, error)

	// @Summary 返回符合条件的操作记录数目
	// @Description 返回符合条件的操作记录数目
	// @Param userid query int   false        "操作人"
	// @Param successful query  bool   false       "操作是否成功"
	// @Param types query   string   false     "操作类型"
	// @Param content_like query   string   false     "描述包含的字符"
	// @Param begin_at query   time.Time   false     "开始时间"
	// @Param end_at query   time.Time   false     "结束时间"
	// @Accept  json
	// @Produce  json
	// @Router /oplog/count [get]
	// @Success 200 {object} int
	Count(ctx context.Context, userid []int64, successful sql.NullBool, types []string, contentLike string, beginAt, endAt time.Time) (int64, error)

	// @Summary 返回符合条件的操作记录数目
	// @Description 返回符合条件的操作记录数目
	// @Param userid query int   false        "操作人"
	// @Param successful query  bool   false       "操作是否成功"
	// @Param types query   string   false     "操作类型"
	// @Param content_like query   string   false     "描述包含的字符"
	// @Param begin_at query   time.Time   false     "开始时间"
	// @Param end_at query   time.Time   false     "结束时间"
	// @Param offset query   int   false     "offset"
	// @Param limit query   int   false     "limit"
	// @Param sort_by query   string   false     "排序字段"
	// @Accept  json
	// @Produce  json
	// @Router /oplog [get]
	// @Success 200 {array} OperationLog
	List(ctx context.Context, userid []int64, successful sql.NullBool, types []string, contentLike string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error)
}
