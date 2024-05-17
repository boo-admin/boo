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
	"github.com/boo-admin/boo/client"

	"github.com/boo-admin/boo/goutils/importer"
)

func TestUserImport1(t *testing.T) {
	app := app_tests.NewTestApp(t, nil)
	app.Start(t)
	defer app.Stop(t)

	ctx := context.Background()
	pxy, err := client.NewResty(app.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}
	pxy.SetBasicAuth("admin", "admin")

	users := client.NewRemoteUsers(pxy)

	// opts := cmp.Options{
	// 	cmpopts.EquateApproxTime(1 * time.Second),
	// 	// cmpopts.IgnoreFields(notifications.Group{}, "ID", "CreatedAt", "UpdatedAt"),
	// 	// cmpopts.IgnoreFields(Baseline{}, "ID", "CreatedAt", "UpdatedAt"),
	// }

	// var dbnow = app.DbNow()

	urlstr, err := url.JoinPath(app.BaseURL, "users/import")
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

	list, err := users.List(ctx, "", "", 0, 0)
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

	list, err = users.List(ctx, "", "", 0, 0)
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
}

func TestEmployeeImport1(t *testing.T) {
	app := app_tests.NewTestApp(t, nil)
	app.Start(t)
	defer app.Stop(t)

	ctx := context.Background()
	pxy, err := client.NewResty(app.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}
	pxy.SetBasicAuth("admin", "admin")

	users := client.NewRemoteEmployees(pxy)

	// opts := cmp.Options{
	// 	cmpopts.EquateApproxTime(1 * time.Second),
	// 	// cmpopts.IgnoreFields(notifications.Group{}, "ID", "CreatedAt", "UpdatedAt"),
	// 	// cmpopts.IgnoreFields(Baseline{}, "ID", "CreatedAt", "UpdatedAt"),
	// }

	// var dbnow = app.DbNow()

	urlstr, err := url.JoinPath(app.BaseURL, "employees/import")
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

	list, err := users.List(ctx, "", "", 0, 0)
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

	list, err = users.List(ctx, "", "", 0, 0)
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
}
