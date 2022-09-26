package utils

import (
	"html/template"
	"ku-chat/pkg/utils/str"
	"ku-chat/pkg/utils/time"
	"ku-chat/pkg/utils/view"
	"reflect"
	"strings"
)

// GetTemplateFuncMap 获取模版函数
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"DiffForHumans":    time.DiffForHumans,
		"ToDateTimeString": time.ToDateTimeString,
		"Html":             view.Html,
		"RemindName":       view.RemindName,
		"StrLimit":         str.Limit,
		"StrJoin":          strings.Join,
	}
}

// StructToMap 结构体转换成字典
func StructToMap(s interface{}) map[string]interface{} {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	data := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}
