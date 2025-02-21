package booclient

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
	Mobile = CustomField{
		ID:    "mobilephone",
		Name:  "手机",
		Alias: []string{"移动电话", "电话", "手机号", "手机号码"},
	}
	Telephone = CustomField{
		ID:    "telephone",
		Name:  "座机",
		Alias: []string{"固定电话", "座机号"},
	}
	IsSupporter = CustomField{
		ID:   "is_supporter",
		Name: "是否为支持人员",
	}

	DefaultUserFields = []CustomField{
		WhiteAddressList,
		WelcomeURL,
		Mobile,
		Telephone,
		Email,
	}

	DefaultEmployeeFields = []CustomField{
		// WhiteAddressList,
		// WelcomeURL,
		Mobile,
		Telephone,
		Email,
		IsSupporter,
	}
)

type CustomField struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// 主要定义导入时的字段别名
	Alias        []string `json:"alias,omitempty"`
	DefaultValue string   `json:"default,omitempty"`
}
