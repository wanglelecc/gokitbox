package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	uni   *ut.UniversalTranslator
	trans ut.Translator

	chineseMobileRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

	initOnce sync.Once
	initErr  error
)

// Init 初始化 gin 校验器，通过 sync.Once 保证全局只执行一次，并发安全。
func Init() error {
	initOnce.Do(func() {
		initErr = doInit()
	})
	return initErr
}

func doInit() error {
	zhLocale := zh.New()
	uni = ut.New(zhLocale, zhLocale)

	var ok bool
	trans, ok = uni.GetTranslator("zh")
	if !ok {
		return fmt.Errorf("validator: failed to get zh translator")
	}

	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return fmt.Errorf("validator: unexpected engine type %T, expected *validator.Validate", binding.Validator.Engine())
	}

	if err := v.RegisterValidation("chineseMobile", chineseMobile); err != nil {
		return fmt.Errorf("validator: register chineseMobile: %w", err)
	}

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("label"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	if err := zh_translations.RegisterDefaultTranslations(v, trans); err != nil {
		return fmt.Errorf("validator: register zh translations: %w", err)
	}

	if err := v.RegisterTranslation("chineseMobile", trans, registerTranslator("chineseMobile", "{0}格式不正确"), translate); err != nil {
		return fmt.Errorf("validator: register chineseMobile translation: %w", err)
	}

	return nil
}

// Translate 翻译校验错误信息。若 Init() 失败，返回空 map。
func Translate(err error) map[string][]string {
	result := make(map[string][]string)
	// 调用 Init() 确保 trans 已初始化，避免并发窗口内读到 nil trans
	if Init() != nil {
		return result
	}
	if errors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errors {
			result[e.StructField()] = append(result[e.StructField()], e.Translate(trans))
		}
	}
	return result
}

// registerTranslator 为自定义字段添加翻译功能
func registerTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		return trans.Add(tag, msg, false)
	}
}

// translate 自定义字段的翻译方法
func translate(trans ut.Translator, fe validator.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field())
	if err != nil {
		return fe.Error()
	}
	return msg
}

// chineseMobile 国内手机号校验
func chineseMobile(fl validator.FieldLevel) bool {
	return chineseMobileRegex.MatchString(fl.Field().String())
}
