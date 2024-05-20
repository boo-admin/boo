package boo

import (
	"context"
	"database/sql"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/services/users"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"
)

type Server struct {
	Logger    *slog.Logger
	Params    map[string]string
	Factory   *gobatis.SessionFactory
	ToRealDir func(context.Context, string) string

	OperationLogger  users.OperationLogger
	OperationQueryer client.OperationQueryer
	Departments      client.Departments
	Users            users.Users
	Employees        users.Employees
}

func NewServer(logger *slog.Logger, params map[string]string, toRealDir func(context.Context, string) string) (*Server, error) {
	srv := &Server{
		Logger:    logger,
		Params:    params,
		ToRealDir: toRealDir,
	}
	dbFactory, err := NewDbFactory(logger, params)
	if err != nil {
		return nil, err
	}
	srv.Factory = dbFactory

	err = RunMigrations(context.Background(), dbFactory.DriverName(), dbFactory.DB().(*sql.DB), params["db.reset_db"] == "true")
	if err != nil {
		return nil, err
	}

	session := dbFactory.SessionReference()
	userDao := users.NewUserDao(session)

	srv.OperationLogger = users.NewOperationLogger(dbFactory)
	operationQueryer, err := users.NewOperationQueryer(params, toRealDir, session,
		func(ctx context.Context, id int64) (string, error) {
			user, err := userDao.FindByID(ctx, id)
			if err != nil {
				return "", err
			}
			return user.Nickname, nil
		})
	if err != nil {
		return nil, err
	}
	srv.OperationQueryer = operationQueryer

	departments, err := users.NewDepartments(logger, params, dbFactory, srv.OperationLogger, toRealDir)
	if err != nil {
		return nil, err
	}
	srv.Departments = departments

	usvc, err := users.NewUsers(logger, params, dbFactory, srv.OperationLogger, toRealDir)
	if err != nil {
		return nil, err
	}
	srv.Users = usvc

	employeeSvc, err := users.NewEmployees(logger, params, dbFactory, srv.OperationLogger, toRealDir)
	if err != nil {
		return nil, err
	}
	srv.Employees = employeeSvc
	return srv, nil
}

func NewDbFactory(logger *slog.Logger, params map[string]string) (*gobatis.SessionFactory, error) {
	driverName := params["db.drv"]
	sourceName := params["db.url"]
	dbConn, err := sql.Open(driverName, sourceName)
	if err != nil {
		return nil, err
	}
	tagSplit := gobatis.SplitXORM

	return gobatis.New(&gobatis.Config{
		Tracer:    client.NewSQLTracer(logger),
		TagPrefix: tagSplit.Prefix,
		TagMapper: tagSplit.Split,
		// Constants:  constants,
		DriverName: driverName,
		DB:         dbConn,
		XMLPaths: []string{
			"gobatis",
		},
	})
}
