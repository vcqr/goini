package goini

import (
	"regexp"
)

const (
	FirstSection   string = `(?U)(\[.*\])`    // 节匹配 只解析第一个符合规范的节名
	Section        string = `\s+|\[|\]|\*+`   // 节匹配
	Node           string = `.*=.*`           // 节点匹配
	Separator      string = `(\.\.+)`         // key中是否含有连续的分隔符
	QuotationStart string = `(?U)(".*"|'.*')` // 获取引号中的内容，只匹配一次
	QuotationEnd   string = `"|'`             // 匹配单或双引号
)

var (
	rxFirstSection   = regexp.MustCompile(FirstSection)
	rxSection        = regexp.MustCompile(Section)
	rxNode           = regexp.MustCompile(Node)
	rxSeparator      = regexp.MustCompile(Separator)
	rxQuotationStart = regexp.MustCompile(QuotationStart)
	rxQuotationEnd   = regexp.MustCompile(QuotationEnd)
)
