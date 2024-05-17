package users

import (
	"context"
	"database/sql"
	"os"
	"time"

	"github.com/boo-admin/boo/client"
	gobatis "github.com/runner-mei/GoBatis"
)

var enableOplog = false

type OperationLogger interface {
	WithTx(tx gobatis.DBRunner) OperationLogger
	LogRecord(ctx context.Context, ol *OperationLog) error
}

type operationLogger struct {
	dao OperationLogDao
}

func (logger operationLogger) Tx(tx *gobatis.Tx) OperationLogger {
	if tx == nil {
		return logger
	}
	return operationLogger{dao: NewOperationLogDao(tx.SessionReference())}
}

func (logger operationLogger) WithTx(tx gobatis.DBRunner) OperationLogger {
	if tx == nil {
		return logger
	}
	return operationLogger{dao: logger.dao.WithDB(tx)}
}

func (logger operationLogger) LogRecord(ctx context.Context, ol *OperationLog) error {
	return logger.dao.Insert(ctx, ol)
}

type operationQueryer struct {
	names map[string]OperationLogLocaleConfig
	dao   OperationLogDao
}

func (queryer operationQueryer) GetLocales(ctx context.Context) (map[string]OperationLogLocaleConfig, error) {
	return queryer.names, nil
}

func (queryer operationQueryer) toTypeTilte(ctx context.Context, typeName string) string {
	s, ok := queryer.names[typeName]
	if !ok && s.Title == "" {
		return typeName
	}
	return s.Title
}

func (queryer operationQueryer) Count(ctx context.Context, userid []int64, successful sql.NullBool, typeList []string, content string, beginAt, endAt time.Time) (int64, error) {
	return queryer.dao.Count(ctx, userid, successful, typeList, content, TimeRange{Start: beginAt, End: endAt})
}

func (queryer operationQueryer) List(ctx context.Context, userid []int64, successful sql.NullBool, typeList []string, content string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error) {
	items, err := queryer.dao.List(ctx, userid, successful, typeList, content, TimeRange{Start: beginAt, End: endAt}, offset, limit, sortBy)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].TypeTitle = queryer.toTypeTilte(ctx, items[idx].Type)
	}
	return items, nil
}

func LoadOperationLogLocaleConfig(params map[string]string,
	toRealDir func(context.Context, string) string) (map[string]OperationLogLocaleConfig, error) {
	filename := toRealDir(context.Background(), "@conf/operation_logs.zh.json")
	customFilename := toRealDir(context.Background(), "@data/conf/operation_logs.zh.json")

	var cfg map[string]OperationLogLocaleConfig
	err := client.FromHjsonFile(filename, &cfg)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var customCfg map[string]OperationLogLocaleConfig
	err = client.FromHjsonFile(customFilename, &customCfg)
	if err != nil {
		if !os.IsNotExist(err) {
			return map[string]OperationLogLocaleConfig{}, nil
		}
	} else if len(cfg) == 0 {
		cfg = customCfg
	} else {
		for key, newValue := range customCfg {
			oldValue, ok := cfg[key]
			if !ok {
				cfg[key] = newValue
				continue
			}
			if newValue.Title != "" {
				oldValue.Title = newValue.Title
			}

			if len(oldValue.Fields) == 0 {
				oldValue.Fields = newValue.Fields
			} else {
				for k, v := range newValue.Fields {
					oldValue.Fields[k] = v
				}
			}
			cfg[key] = newValue
		}
	}
	return cfg, nil
}

func NewOperationQueryer(params map[string]string,
	toRealDir func(context.Context, string) string,
	session gobatis.SqlSession,
	findUsernameByID func(ctx context.Context, id int64) (string, error)) (client.OperationQueryer, error) {
	names, err := LoadOperationLogLocaleConfig(params, toRealDir)
	if err != nil {
		return nil, err
	}
	return operationQueryer{names: names, dao: NewOperationLogDao(session)}, nil
}

func NewOperationLogger(session gobatis.SqlSession) OperationLogger {
	return operationLogger{dao: NewOperationLogDao(session)}
}
