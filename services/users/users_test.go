package users_test

import (
	"context"
	"testing"
	"time"

	"github.com/boo-admin/boo/app_tests"
	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/validation"
	"github.com/boo-admin/boo/services/users"

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


		cmpopts.IgnoreFields(booclient.Role{}, "ID", "CreatedAt", "UpdatedAt"),
		cmpopts.IgnoreFields(booclient.TagData{}, "ID"),
		cmpopts.IgnoreFields(booclient.Department{}, "ID", "CreatedAt", "UpdatedAt"),
		cmpopts.IgnoreFields(users.UserTag{}, "ID", "CreatedAt", "UpdatedAt"),
	}

	ctx := context.Background()
	pxy, err := booclient.NewResty(app.BaseURL())
	if err != nil {
		t.Error(err)
		return
	}
	pxy.SetBasicAuth("admin", "admin")

	departments := booclient.NewRemoteDepartments(pxy)
	var dbnow = app.DbNow()
	departmentID, err := departments.Create(ctx, &booclient.Department{
		UUID:      "abc",
		Name:      "abc",
		UpdatedAt: dbnow,
		CreatedAt: dbnow,
	})
	if err != nil {
		t.Error(err)
		return
	}

	users := booclient.NewRemoteUsers(pxy)

	data := &booclient.User{
		Name:         "abc",
		Nickname:     "测试用户名1",
		DepartmentID: departmentID,
		Description:  "stsres",
		// Source: "default",
		Disabled: false,
		Fields: map[string]interface{}{
			booclient.Mobile.ID: "1334567",
			booclient.Email.ID: "a@b.com",
		},

		UpdatedAt: dbnow,
		CreatedAt: dbnow,

		Tags: []booclient.TagData{
			{UUID: "uid1"},
			{Title: "title2"},
			{UUID: "uid3", Title: "title3"},
		},

		Roles: []booclient.Role{
			{UUID: "ruid1"},
			{Title: "rtitle2"},
			{UUID: "ruid3", Title: "rtitle3"},
		},
	}

	userid, err := users.Create(ctx, data)
	if err == nil {
		t.Error("want error got ok")
		return
	} else if !errors.Is(err, errors.ErrValidationError) {
		t.Error(err)
		t.Errorf("%#v", err)
		return
	} else {
		ok, errList := validation.ToValidationErrors(err)
		if !ok {
			t.Error(err)
			t.Errorf("%#v", err)
			return
		}

		if errList[0].Key != "Password" {
			t.Error(errList[0].Code, errList[0].Key, errList[0].Message)
		}
	}

	data.Password = "asdf#1=$AuH@*&"
	userid, err = users.Create(ctx, data)
	if err != nil {
		t.Error(err)
		return
	}

	actual, err := users.FindByID(ctx, userid, "*")
	if err != nil {
		t.Error(err)
		return
	}

 	data.Department = &booclient.Department{
                       UUID:      "abc",
                       Name:      "abc",
                       Fields:    map[string]interface{}{},
               }

  for idx := range data.Tags {
  	if data.Tags[idx].UUID == "" {
  		data.Tags[idx].UUID = actual.Tags[idx].UUID
  	}
  	if data.Tags[idx].Title == "" {
  		data.Tags[idx].Title = actual.Tags[idx].Title
  	}
  }
  for idx := range data.Roles {
  	if data.Roles[idx].UUID == "" {
  		data.Roles[idx].UUID = actual.Roles[idx].UUID
  	}
  	if data.Roles[idx].Title == "" {
  		data.Roles[idx].Title = actual.Roles[idx].Title
  	}
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
		booclient.Email.ID: "c@b.com",
	}

	err = users.UpdateByID(ctx, userid, data, booclient.UpdateModeSkip)
	if err != nil {
		t.Error(err)
		return
	}

	actual, err = users.FindByID(ctx, userid, "*")
	if err != nil {
		t.Error(err)
		return
	}

	data.Fields = map[string]interface{}{
		booclient.Mobile.ID: "1334567",
		booclient.Email.ID: "c@b.com",
	}
	if !cmp.Equal(data, actual, opts...) {
		diff := cmp.Diff(data, actual, opts...)
		t.Error(diff)
	}

	list, err := users.List(ctx, 0, "", "", "", booclient.None, []string{"*"}, "", 0, 0)
	if err != nil {
		t.Error(err)
		return
	}

	if !cmp.Equal(data, &list[0], opts...) {
		diff := cmp.Diff(data, &list[0], opts...)
		t.Error(diff)
	}

	count, err := users.Count(ctx, 0, "", "", "", booclient.None)
	if err != nil {
		t.Error(err)
		return
	}
	if count != 1 {
		t.Error("want 1 got ", count)
	}

	err = users.DeleteByID(ctx, userid, true)
	if err != nil {
		t.Error(err)
		return
	}

	count, err = users.Count(ctx, 0, "", "", "", booclient.None)
	if err != nil {
		t.Error(err)
		return
	}
	if count != 0 {
		t.Error("want 0 got ", count)
	}
}
