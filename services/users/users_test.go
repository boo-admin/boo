package users_test

import (
	"context"
	"testing"
	"time"
	"strings"

	"github.com/boo-admin/boo/app_tests"
	"github.com/boo-admin/boo/client"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestUser1(t *testing.T) {
	app := app_tests.NewTestApp(t, nil)
	app.Start(t)
	defer app.Stop(t)

	opts := cmp.Options{
		cmpopts.EquateApproxTime(1 * time.Second),
		// cmpopts.IgnoreFields(notifications.Group{}, "ID", "CreatedAt", "UpdatedAt"),
		// cmpopts.IgnoreFields(BackupRule{}, "ID", "CreatedAt", "UpdatedAt"),
	}

	ctx := context.Background()
	pxy, err := client.NewResty(app.BaseURL())
	if err != nil {
		t.Error(err)
		return
	}
	pxy.SetBasicAuth("admin", "admin")

	departments := client.NewRemoteDepartments(pxy)
	var dbnow = app.DbNow()
	departmentID, err := departments.Create(ctx, &client.Department{
		UUID:      "abc",
		Name:      "abc",
		UpdatedAt: dbnow,
		CreatedAt: dbnow,
	})
	if err != nil {
		t.Error(err)
		return
	}

	users := client.NewRemoteUsers(pxy)

	data := &client.User{
		Name:         "abc",
		Nickname:     "测试用户名1",
		DepartmentID: departmentID,
		Description:  "stsres",
		// Source: "default",
		Disabled: false,
		Fields: map[string]interface{}{
			client.Phone.ID: "1334567",
			client.Email.ID: "a@b.com",
		},

		UpdatedAt: dbnow,
		CreatedAt: dbnow,
	}

	userid, err := users.Create(ctx, data)
	if err == nil {
		t.Error("want error got ok")
		return
	} else if !strings.Contains(err.Error(), "abc") {
		t.Error(err)
		t.Errorf("%#v", err)
		return
	}

	data.Password =     "asdf#1=$AuH@*&"
	userid, err = users.Create(ctx, data)
	if err != nil {
		t.Error(err)
		return
	}

	actual, err := users.FindByID(ctx, userid)
	if err != nil {
		t.Error(err)
		return
	}

	data.Password = "******"
	data.ID = userid
	if !cmp.Equal(data, actual, opts...) {
		diff := cmp.Diff(data, actual, opts...)
		t.Error(diff)
	}

	data.ID = userid
	data.Nickname = "abcd"
	data.Description = "abcd"
	data.Fields = map[string]interface{}{
		client.Email.ID: "c@b.com",
	}

	err = users.UpdateByID(ctx, userid, data)
	if err != nil {
		t.Error(err)
		return
	}

	actual, err = users.FindByID(ctx, userid)
	if err != nil {
		t.Error(err)
		return
	}

	data.Fields = map[string]interface{}{
		client.Phone.ID: "1334567",
		client.Email.ID: "c@b.com",
	}
	if !cmp.Equal(data, actual, opts...) {
		diff := cmp.Diff(data, actual, opts...)
		t.Error(diff)
	}

	list, err := users.List(ctx,0, "", "", 0, 0)
	if err != nil {
		t.Error(err)
		return
	}

	if !cmp.Equal(data, &list[0], opts...) {
		diff := cmp.Diff(data, &list[0], opts...)
		t.Error(diff)
	}

	count, err := users.Count(ctx, 0, "")
	if err != nil {
		t.Error(err)
		return
	}
	if count != 1 {
		t.Error("want 1 got ", count)
	}

	err = users.DeleteByID(ctx, userid)
	if err != nil {
		t.Error(err)
		return
	}

	count, err = users.Count(ctx, 0, "")
	if err != nil {
		t.Error(err)
		return
	}
	if count != 0 {
		t.Error("want 0 got ", count)
	}
}
