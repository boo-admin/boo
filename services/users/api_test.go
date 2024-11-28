package users_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/boo-admin/boo/app_tests"
	"github.com/boo-admin/boo/booclient"

	"github.com/boo-admin/boo/goutils/importer"
)

func TestUserImport1(t *testing.T) {
	app := app_tests.NewTestApp(t, nil)
	app.Start(t)
	defer app.Stop(t)

	ctx := context.Background()
	pxy, err := booclient.NewResty(app.BaseURL())
	if err != nil {
		t.Error(err)
		return
	}
	pxy.SetBasicAuth("admin", "admin")

	users := booclient.NewRemoteUsers(pxy)

	// opts := cmp.Options{
	// 	cmpopts.EquateApproxTime(1 * time.Second),
	// 	// cmpopts.IgnoreFields(notifications.Group{}, "ID", "CreatedAt", "UpdatedAt"),
	// 	// cmpopts.IgnoreFields(Baseline{}, "ID", "CreatedAt", "UpdatedAt"),
	// }

	// var dbnow = app.DbNow()

	urlstr, err := url.JoinPath(app.BaseURL(), "users/import")
	if err != nil {
		t.Error(err)
		return
	}
	urlstr = urlstr + "?department_auto_create=true"

	reader, err := os.Open("./test.xlsx")
	if err != nil {
		t.Error(err)
		return
	}
	defer reader.Close()

	// response, err := importer.UploadFile(nil, urlstr, nil, "file", "test.xlsx", reader)

	request, err := importer.NewUploadRequest(urlstr, nil, "file", "test.xlsx", reader)
	if err != nil {
		t.Error(err)
		return
	}
	request.SetBasicAuth("admin", "admin")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err)
		return
	}

	bs, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		t.Error(response.Status)
		if len(bs) > 0 {
			t.Error(string(bs))
		}
		return
	}

	list, err := users.List(ctx, 0, "", "", "", booclient.None, []string{"tags"}, "", 0, 0)
	if err != nil {
		t.Error(err)
		return
	}

	var usernames []string
	var phones []string
	for idx := range list {
		usernames = append(usernames, list[idx].Name)
		phones = append(phones, list[idx].GetPhone())
	}
	exceptedUsernames := []string{"王源", "周宏", "李琦", "杨尚", "王晶"}
	if !reflect.DeepEqual(usernames, exceptedUsernames) {
		t.Error(usernames)
		t.Error(exceptedUsernames)
	}

	assetTag := func(t testing.TB, name string, tags []string) {
		for idx := range list {
			if list[idx].Name == name {
				if len(tags) == 0 && len(list[idx].Tags) == 0 {
					return
				}

				var actualtags []string
				for _, tag := range list[idx].Tags {
					actualtags = append(actualtags, tag.Title)
				}
				if !reflect.DeepEqual(actualtags, tags) {
					t.Error(name, actualtags, tags)
				}
				return
			}
		}
		t.Error("not found")
	}

	assetTag(t, "王源", []string{"测1","测2"})
	assetTag(t, "周宏", []string{})
	assetTag(t, "李琦", []string{"测2"})
	assetTag(t, "杨尚", []string{})
	assetTag(t, "王晶", []string{})

	exceptedPhones := []string{"14228883500", "14228883600", "14228883400", "14228883700", "14228883800"}
	if !reflect.DeepEqual(phones, exceptedPhones) {
		t.Error(phones)
		t.Error(exceptedPhones)
	}

	reader, err = os.Open("./updatetest.xlsx")
	if err != nil {
		t.Error(err)
		return
	}
	defer reader.Close()

	request, err = importer.NewUploadRequest(urlstr, nil, "file", "updatetest.xlsx", reader)
	if err != nil {
		t.Error(err)
		return
	}
	request.SetBasicAuth("admin", "admin")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err)
		return
	}

	bs, _ = ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		t.Error(response.Status)
		if len(bs) > 0 {
			t.Error(string(bs))
		}
		return
	}

	list, err = users.List(ctx, 0, "", "", "", booclient.None, []string{"tags"}, "", 0, 0)
	if err != nil {
		t.Error(err)
		return
	}

	usernames = nil
	phones = nil
	for idx := range list {
		usernames = append(usernames, list[idx].Name)
		phones = append(phones, list[idx].GetPhone())
	}
	exceptedUsernames = []string{"王源", "周宏", "李琦", "杨尚", "王晶"}
	if !reflect.DeepEqual(usernames, exceptedUsernames) {
		t.Error(usernames)
		t.Error(exceptedUsernames)
	}

	exceptedPhones = []string{"14228883500update", "14228883600update", "14228883400update", "14228883700update", "14228883800update"}
	if !reflect.DeepEqual(phones, exceptedPhones) {
		t.Error(phones)
		t.Error(exceptedPhones)
	}

	assetTag(t, "王源", []string{"测1", "测2","标1"})
	assetTag(t, "周宏", []string{})
	assetTag(t, "李琦", []string{"测2", "标签2"})
	assetTag(t, "杨尚", []string{})
	assetTag(t, "王晶", []string{})
}

func TestEmployeeImport1(t *testing.T) {
	app := app_tests.NewTestApp(t, nil)
	app.Start(t)
	defer app.Stop(t)

	ctx := context.Background()
	pxy, err := booclient.NewResty(app.BaseURL())
	if err != nil {
		t.Error(err)
		return
	}
	pxy.SetBasicAuth("admin", "admin")

	users := booclient.NewRemoteEmployees(pxy)

	// opts := cmp.Options{
	// 	cmpopts.EquateApproxTime(1 * time.Second),
	// 	// cmpopts.IgnoreFields(notifications.Group{}, "ID", "CreatedAt", "UpdatedAt"),
	// 	// cmpopts.IgnoreFields(Baseline{}, "ID", "CreatedAt", "UpdatedAt"),
	// }

	// var dbnow = app.DbNow()

	urlstr, err := url.JoinPath(app.BaseURL(), "employees/import")
	if err != nil {
		t.Error(err)
		return
	}
	urlstr = urlstr + "?department_auto_create=true"

	reader, err := os.Open("./test.xlsx")
	if err != nil {
		t.Error(err)
		return
	}
	defer reader.Close()

	// response, err := importer.UploadFile(nil, urlstr, nil, "file", "test.xlsx", reader)

	request, err := importer.NewUploadRequest(urlstr, nil, "file", "test.xlsx", reader)
	if err != nil {
		t.Error(err)
		return
	}
	request.SetBasicAuth("admin", "admin")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err)
		return
	}

	bs, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		t.Error(response.Status)
		if len(bs) > 0 {
			t.Error(string(bs))
		}
		return
	}

	list, err := users.List(ctx, 0, "", "", booclient.None, []string{"*"}, "", 0, 0)
	if err != nil {
		t.Error(err)
		return
	}


	assetTag := func(t testing.TB, name string, tags []string) {
		for idx := range list {
			if list[idx].Name == name {
				if len(tags) == 0 && len(list[idx].Tags) == 0 {
					return
				}

				var actualtags []string
				for _, tag := range list[idx].Tags {
					actualtags = append(actualtags, tag.Title)
				}
				if !reflect.DeepEqual(actualtags, tags) {
					t.Error(name, actualtags, tags)
				}
				return
			}
		}
		t.Error("not found")
	}

	assetTag(t, "王源", []string{"测1","测2"})
	assetTag(t, "周宏", []string{})
	assetTag(t, "李琦", []string{"测2"})
	assetTag(t, "杨尚", []string{})
	assetTag(t, "王晶", []string{})

	var usernames []string
	var phones []string
	for idx := range list {
		usernames = append(usernames, list[idx].Name)
		phones = append(phones, list[idx].GetPhone())
	}
	exceptedUsernames := []string{"王源", "周宏", "李琦", "杨尚", "王晶"}
	if !reflect.DeepEqual(usernames, exceptedUsernames) {
		t.Error(usernames)
		t.Error(exceptedUsernames)
	}

	exceptedPhones := []string{"14228883500", "14228883600", "14228883400", "14228883700", "14228883800"}
	if !reflect.DeepEqual(phones, exceptedPhones) {
		t.Error(phones)
		t.Error(exceptedPhones)
	}

	reader, err = os.Open("./updatetest.xlsx")
	if err != nil {
		t.Error(err)
		return
	}
	defer reader.Close()

	request, err = importer.NewUploadRequest(urlstr, nil, "file", "updatetest.xlsx", reader)
	if err != nil {
		t.Error(err)
		return
	}
	request.SetBasicAuth("admin", "admin")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err)
		return
	}

	bs, _ = ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		t.Error(response.Status)
		if len(bs) > 0 {
			t.Error(string(bs))
		}
		return
	}

	list, err = users.List(ctx, 0, "", "", booclient.None, []string{"*"}, "", 0, 0)
	if err != nil {
		t.Error(err)
		return
	}

	usernames = nil
	phones = nil
	for idx := range list {
		usernames = append(usernames, list[idx].Name)
		phones = append(phones, list[idx].GetPhone())
	}
	exceptedUsernames = []string{"王源", "周宏", "李琦", "杨尚", "王晶"}
	if !reflect.DeepEqual(usernames, exceptedUsernames) {
		t.Error(usernames)
		t.Error(exceptedUsernames)
	}

	exceptedPhones = []string{"14228883500update", "14228883600update", "14228883400update", "14228883700update", "14228883800update"}
	if !reflect.DeepEqual(phones, exceptedPhones) {
		t.Error(phones)
		t.Error(exceptedPhones)
	}

	
	assetTag(t, "王源", []string{"测1", "测2","标1"})
	assetTag(t, "周宏", []string{})
	assetTag(t, "李琦", []string{"测2", "标签2"})
	assetTag(t, "杨尚", []string{})
	assetTag(t, "王晶", []string{})
}
