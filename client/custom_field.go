package client

var (
	WhiteAddressList = CustomField{
		ID:   "white_address_list",
		Name: "登录IP",
	}
	WelcomeURL = CustomField{
		ID:   "welcome",
		Name: "首页",
	}
	Email = CustomField{
		ID:   "email",
		Name: "邮箱",
	}
	Phone = CustomField{
		ID:    "phone",
		Name:  "电话",
		Alias: []string{"手机"},
	}

	DefaultFields = []CustomField{
		WhiteAddressList,
		WelcomeURL,
		Phone,
		Email,
	}
)

type CustomField struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// 主要定义导入时的字段别名
	Alias        []string `json:"alias,omitempty"`
	DefaultValue string   `json:"default,omitempty"`
}
