
set root_dir=%~dp0


@set is_tools=
@set is_generate=
@set is_clean=
@set is_compile=
@set is_test=
@set is_install=
@set is_goose=
@set is_lint=

:next-arg

@if "%1"=="" goto args-done
@if /i "%1"=="tools"        set is_tools=1&goto arg-ok
@if /i "%1"=="gen"          set is_generate=1&goto arg-ok
@if /i "%1"=="genall"       set is_generate=1&goto arg-ok
@if /i "%1"=="clean"        set is_clean=1&goto arg-ok
@if /i "%1"=="test"         set is_test=1&goto arg-ok
@if /i "%1"=="goose"        set is_goose=1 & shift & goto goose_begin
@if /i "%1"=="lint"         set is_lint=1 & shift & goto lint_begin
@if /i "%1"=="compile"      set is_compile=1&goto arg-ok
@if /i "%1"=="install"      set is_compile=1&set is_install=1&goto arg-ok
@if /i "%1"=="all"          set is_tools=1&set is_clean=1&set is_generate=1&set is_test=1&set is_compile=1&set is_install=1&goto arg-ok

@goto help

:arg-ok
@shift
@goto next-arg
:args-done

@if not defined is_tools goto tool_ok
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/runner-mei/GoBatis/cmd/gobatis
go install github.com/runner-mei/gogen/v2/cmd/gogenv2
go install github.com/swaggo/swag/cmd/swag@latest
:tool_ok

set GOGEN_PLUGIN=echo
set GOGEN_ERRORS=github.com/boo-admin/boo/errors
set GOGEN_IMPORTS=github.com/boo-admin/boo,booclient "github.com/boo-admin/boo/client",github.com/boo-admin/boo/engine/echofunctions
set GOGEN_HTTPCODEWITH=errors.HttpCodeWith
set GOGEN_TOJSONERROR=errors.ToEncodeError
set GOGEN_BADARGUMENT=errors.NewBadArgument
set GOGEN_CONTEXT_GETTER=echofunctions.GetContext(ctx)

@echo "GO111MODULE=%GO111MODULE%"
@echo "GOROOT=%GOROOT%"
@echo "GOPATH=%GOPATH%"
@echo "CI_NAME=%CI_NAME%"
@echo "GOGEN_PLUGIN=%GOGEN_PLUGIN%"
@echo "GOGEN_ERRORS=%GOGEN_ERRORS%"
@echo "GOGEN_IMPORTS=%GOGEN_IMPORTS%"
@echo "GOGEN_HTTPCODEWITH=%GOGEN_HTTPCODEWITH%"
@echo "GOGEN_TOJSONERROR=%GOGEN_TOJSONERROR%"
@echo "GOGEN_BADARGUMENT=%GOGEN_BADARGUMENT%"
@echo "GOGEN_CONTEXT_GETTER=%GOGEN_CONTEXT_GETTER%"



@if not defined is_clean goto clean_ok
del /s .\*-gen.go
del /s .\*gobatis.go
del /s .\client\*-gen.go
del /s .\client\*gobatis.go
del /s .\services\users\*-gen.go
del /s .\services\users\*gobatis.go
:clean_ok

@if not defined is_generate goto generate_ok

mkdir .\services\docs
go generate ./client
go generate ./services/users
go generate .
:generate_ok


@if not defined is_test goto test_ok
go test .
@if %ERRORLEVEL% NEQ 0 (
	@echo ############
	@echo test fail...
	goto :eof
)
go test -v ./services/users
@if %ERRORLEVEL% NEQ 0 (
	@echo ############
	@echo test fail...
	goto :eof
)
@echo test ok
:test_ok


@if not defined is_compile goto compile_ok
@pushd .\cmd\boo
go build
@if %ERRORLEVEL% NEQ 0 (
	@popd

	@echo #############
	@echo build fail...
	@goto :eof
)
@popd
@echo build ok
:compile_ok


@if not defined is_install goto install_ok

:install_ok

:goose_begin
@if not defined is_goose goto goose_ok
goose -dir migrations\postgres create %1 %2 sql
@goto :eof
:goose_ok


:lint_begin
@if not defined is_lint goto lint_ok
golangci-lint run --tests=false %1 %2
@goto :eof
:lint_ok


@goto :eof
:help
@echo 使用方法如下  build  命令
@echo  tools    安装相关工具
@echo  gen      生成编译所需要的代码
@echo  genall   同 gen 一样, 它是一个 gen 别名
@echo  clean    清空 gen 命令生成的代码
@echo  test     运行单无测试
@echo  compile  编译程序
@echo  all      运行上面所有命令
@echo   
@echo  开发过程中用的命令
@echo  lint     运行代码检查
@echo  goose    生成数据库初始化 sql 文件
@goto :eof
:help_ok