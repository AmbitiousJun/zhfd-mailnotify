package main

import (
	"fmt"
	"strings"
)

// HtmlBuilder 将参数动态加载到模板中的构造器
type HtmlBuilder struct {
	string
	result string
}

// NewHtmlBuilder 根据 html 模板初始化一个构造器
func NewHtmlBuilder(template []byte) *HtmlBuilder {
	return &HtmlBuilder{
		string: string(template),
		result: string(template),
	}
}

// Set 设置参数到模板中
func (hb *HtmlBuilder) Set(key, value string) *HtmlBuilder {
	hb.result = strings.ReplaceAll(hb.result, fmt.Sprintf("${%s}", key), value)
	return hb
}

// Build 将构造好的 Html 文本返回
func (hb *HtmlBuilder) Build() string {
	defer func() {
		// 将 result 恢复为默认模板
		hb.result = hb.string
	}()
	return hb.result
}
