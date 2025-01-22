package boo

import (
	"context"
	"database/sql"

	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/services/users"
	gobatis "github.com/runner-mei/GoBatis"
)

type Server struct {
	Env       *booclient.Environment
	Params    map[string]string
	Factory   *gobatis.SessionFactory
	ToRealDir func(context.Context, string) string

	OperationLogger  users.OperationLogger
	OperationQueryer booclient.OperationQueryer
	Departments      booclient.Departments
	Users            users.Users
	UserTags         booclient.UserTags
	Roles            booclient.Roles
	Employees        users.Employees
	EmployeeTags     booclient.EmployeeTags
}

func SetAutoMigrations(env *booclient.Environment, value bool) *booclient.Environment {
	env.Config.Set("auto_migrations", value)
	return env
}

func NewServer(env *booclient.Environment) (*Server, error) {
	srv := &Server{
		Env: env,
	}
	dbFactory, err := NewDbFactory(env)
	if err != nil {
		return nil, err
	}
	srv.Factory = dbFactory

	if env.Config.BoolWithDefault("auto_migrations", true) {
		err = RunMigrations(context.Background(), dbFactory.DriverName(), dbFactory.DB().(*sql.DB), env.Config.BoolWithDefault("db.reset_db", false))
		if err != nil {
			return nil, err
		}
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

	userTagSvc, err := users.NewUserTags(env, dbFactory, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.UserTags = userTagSvc

	rsvc, err := users.NewRoles(env, dbFactory, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.Roles = rsvc

	employeeSvc, err := users.NewEmployees(env, dbFactory, usvc, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.Employees = employeeSvc

	employeeTagSvc, err := users.NewEmployeeTags(env, dbFactory, srv.OperationLogger)
	if err != nil {
		return nil, err
	}
	srv.EmployeeTags = employeeTagSvc

	return srv, nil
}

func NewDbFactory(env *booclient.Environment) (*gobatis.SessionFactory, error) {
	driverName := env.Config.StringWithDefault("db.drv", "")
	sourceName := env.Config.StringWithDefault("db.url", "")
	dbConn, err := sql.Open(driverName, sourceName)
	if err != nil {
		return nil, err
	}
	tagSplit := gobatis.SplitXORM

	return gobatis.New(&gobatis.Config{
		Tracer:    booclient.NewSQLTracer(env.Logger.WithGroup("db")),
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
