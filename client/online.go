//go:generate gogenv2 server -ext=.server-gen.go online.go
//go:generate gogenv2 client -ext=.client-gen.go online.go

package client

import (
	"context"
	"time"
)

type OnlineInfo struct {
	UUID      string    `json:"uuid"`
	Username  string    `json:"username"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OnlineQueryer interface {
	// @Summary 返回指定的在线用户
	// @Param uuid path string   true        "会话 ID"
	// @Tags     Auth,Users
	// @Accept   json
	// @Produce  json
	// @Router  /online-users/by_uuid/{uuid} [get]
	// @Success 200 {object}  OnlineInfo
	GetBySessionID(ctx context.Context, uuid string) (*OnlineInfo, error)

	// @Summary 返回在线用户
	// @Tags     Auth,Users
	// @Accept   json
	// @Produce  json
	// @Router   /online-users [get]
	// @Success  200 {array}  OnlineInfo
	List(ctx context.Context) ([]OnlineInfo, error)

	// @Summary 返回在线用户人数
	// @Tags     Auth,Users
	// @Accept   json
	// @Produce  json
	// @Router   /online-users/count [get]
	// @Success  200 {int}  int
	Count(ctx context.Context) (int64, error)

	// @Summary 删除在线用户
	// @Tags     Auth,Users
	// @Param    username path string  true        "用户 ID"
	// @Accept   json
	// @Produce  json
	// @Router   /online-users/by_user/{username} [delete]
	// @Success  200 {string}  string  "返回一个无意义的 'ok'"
	LogoutByUsername(ctx context.Context, username string) error

	// @Summary 删除在线会话
	// @Tags     Auth,Users
	// @Param    uuid path string   true        "会话 ID"
	// @Accept   json
	// @Produce  json
	// @Router   /online-users/by_uuid/{uuid} [delete]
	// @Success 200 {string}  string  "返回一个无意义的 'ok'"
	LogoutBySessionID(ctx context.Context, uuid string) error
}

type LockedUser struct {
	Username  string
	Address   string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LockedUsers interface {
	// @Summary 返回在线用户
	// @Tags     Auth,Users
	// @Accept  json
	// @Produce  json
	// @Router /locked-users [get]
	// @Success 200 {array}  LockedUser
	List(ctx context.Context) ([]LockedUser, error)

	// @Summary 返回在线用户
	// @Tags     Auth,Users
	// @Param username path string  true        "用户名"
	// @Accept  json
	// @Produce  json
	// @Router /locked-users/by_username/{username} [get]
	// @Success 200 {array}  LockedUser
	FindByID(ctx context.Context, username string) (*LockedUser, error)

	// @Summary 按用户名解锁用户
	// @Tags     Auth,Users
	// @Param username path string  true        "用户名"
	// @Accept  json
	// @Produce  json
	// @Router /locked-users/by_username/{username} [delete]
	// @Success 200 {string}  string  "返回一个无意义的 'ok'"
	Unlock(ctx context.Context, username string) error
}
