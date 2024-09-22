package authncore

import (
	"encoding/json"
	"net/http"

	"github.com/boo-admin/boo/errors"
	"github.com/mojocn/base64Captcha"
)

// CaptchaConfig json request body.
type CaptchaConfig struct {
	CaptchaType   string
	DriverAudio   *base64Captcha.DriverAudio
	DriverString  *base64Captcha.DriverString
	DriverChinese *base64Captcha.DriverChinese
	DriverMath    *base64Captcha.DriverMath
	DriverDigit   *base64Captcha.DriverDigit
}

func GenerateCaptcha(store base64Captcha.Store, config CaptchaConfig) (string, string, string, error) {
	var driver base64Captcha.Driver
	switch config.CaptchaType {
	case "audio":
		driver = config.DriverAudio
	case "string":
		driver = config.DriverString.ConvertFonts()
	case "math":
		driver = config.DriverMath.ConvertFonts()
	case "chinese":
		driver = config.DriverChinese.ConvertFonts()
	default:
		driver = config.DriverDigit
	}
	c := base64Captcha.NewCaptcha(driver, store)
	return c.Generate()
}

// base64Captcha create http handler
func GenerateCaptchaHandler(store base64Captcha.Store, config CaptchaConfig) func(w http.ResponseWriter, r *http.Request) {
	if store == nil {
		store = base64Captcha.DefaultMemStore
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id, base64String, _ /* answer */, err := GenerateCaptcha(store, config)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"data":        base64String,
			"captcha_key": id,
			"msg":         "success",
		})
	}
}

var failResponse = map[string]interface{}{"success": false, "data": "验证失败", "msg": "captcha failed"}
var okResponse = map[string]interface{}{"success": true, "data": "验证通过", "msg": "captcha verified"}

func CaptchaCheck(store base64Captcha.Store, counter FailCounter) AuthOption {
	if store == nil {
		store = base64Captcha.DefaultMemStore
	}
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnBeforeLoad(AuthFunc(func(ctx *AuthContext) error {
			if ctx.SkipCaptcha {
				return nil
			}

			errCount := counter.Count(ctx.Request.Username)
			if ctx.Request.CaptchaKey == "" || ctx.Request.CaptchaValue == "" {
				if errCount > 0 {
					return ErrCaptchaMissing
				}
				return nil
			}

			//比较图像验证码
			if !store.Verify(ctx.Request.CaptchaKey, ctx.Request.CaptchaValue, true) {
				return ErrCaptchaKey
			}
			return nil
		}))
		return nil
	})
}

// base64Captcha verify http handler
func CaptchaVerify(store base64Captcha.Store) func(w http.ResponseWriter, r *http.Request) (bool, error) {
	if store == nil {
		store = base64Captcha.DefaultMemStore
	}

	return func(w http.ResponseWriter, r *http.Request) (bool, error) {
		var captchaKey, verifyValue string

		// Parse the body depending on the content type.
		switch r.Header.Get("Content-Type") {
		case "application/x-www-form-urlencoded":
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				return false, errors.Wrap(err, "读参数失败")
			}
			captchaKey = r.FormValue("captcha_key")
			verifyValue = r.FormValue("captcha_value")

		case "multipart/form-data":
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				return false, errors.Wrap(err, "读参数失败")
			}
			captchaKey = r.FormValue("captcha_key")
			verifyValue = r.FormValue("captcha_value")
		case "application/json":
			fallthrough
		case "text/json":
			var form struct {
				CaptchaKey string `json:"captcha_key"`
				Value      string `json:"captcha_value"`
			}

			if r.Body != nil {
				err := json.NewDecoder(r.Body).Decode(&form)
				if err != nil {
					return false, errors.Wrap(err, "读参数失败")
				}
			}

			captchaKey = form.CaptchaKey
			verifyValue = form.Value
		}

		if captchaKey == "" || verifyValue == "" {
			return false, errors.New("参数为空")
		}

		// 比较图像验证码
		return store.Verify(captchaKey, verifyValue, true), nil
	}
}
