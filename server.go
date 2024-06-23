package boo

import (
	"context"
	"database/sql"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/services/users"
	gobatis "github.com/runner-mei/GoBatis"
)

type Server struct {
	Env       *client.Environment
	Params    map[string]string
	Factory   *gobatis.SessionFactory
	ToRealDir func(context.Context, string) string

	OperationLogger  users.OperationLogger
	OperationQueryer client.OperationQueryer
	Departments      client.Departments
	Users            users.Users
	Employees        users.Employees
}

func NewServer(env *client.Environment) (*Server, error) {
	srv := &Server{
		Env: env,
	}
	dbFactory, err := NewDbFactory(env)
	if err != nil {
		return nil, err
	}
	srv.Factory = dbFactory

	err = RunMigrations(context.Background(), dbFactory.DriverName(), dbFactory.DB().(*sql.DB), env.Config.BoolWithDefault("db.reset_db", false))
	if err != nil {
		return nil, err
	}

	session := dbFactory.SessionReference()
	userDao := users.NewUserDao(session)

	srv.OperationLogger = users.NewOperationLogger(dbFactory)
	operationQueryer, err := users.NewOperationQueryer(env, session,
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

	departments, err := users.NewDepartments(env, dbFactory, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.Departments = departments

	usvc, err := users.NewUsers(env, dbFactory, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.Users = usvc

	employeeSvc, err := users.NewEmployees(env, dbFactory, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.Employees = employeeSvc
	return srv, nil
}

func NewDbFactory(env *client.Environment) (*gobatis.SessionFactory, error) {
	driverName := env.Config.StringWithDefault("db.drv", "")
	sourceName := env.Config.StringWithDefault("db.url", "")
	dbConn, err := sql.Open(driverName, sourceName)
	if err != nil {
		return nil, err
	}
	tagSplit := gobatis.SplitXORM

	return gobatis.New(&gobatis.Config{
		Tracer:    client.NewSQLTracer(env.Logger.WithGroup("db")),
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
