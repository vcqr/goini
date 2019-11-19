package goini

import (
	"fmt"
	"strings"
)

// 记录当前行数据
type TomlLineNode struct {
	KeyName  string
	Arr      []map[string]interface{}
	ArrIndex int
	Operate  string
}

var tomlLineNode = TomlLineNode{
	KeyName: defaultName,
}

// 设置全局map表内容
func setTomlGlobalMapValue(keyName string, value interface{}) {
	//设置属性
	setProperty(keyName, value)
	//设置值
	property[keyName] = value
}

func parseTomlLine(rowB []byte) {
	// 去除空白，空格等字符
	rowStr := string(rowB)
	trimStr := strings.TrimSpace(rowStr)
	if trimStr == "" {
		return
	}

	// 解析注释行
	if strings.HasPrefix(trimStr, "#") {
		return
	}

	if strings.HasPrefix(trimStr, "[[") {
		strArr := findSliceString(trimStr)
		if len(strArr) > 0 {
			tempKey := strArr[0]
			keyName := tempKey[2 : len(tempKey)-2]

			tomlLineNode.Arr = append(tomlLineNode.Arr, make(map[string]interface{}))

			if keyName == tomlLineNode.KeyName {
				// 初始化map
				tomlLineNode.ArrIndex++
				return
			}

			tomlLineNode.KeyName = keyName
			tomlLineNode.Operate = "array"
			tomlLineNode.ArrIndex = 0

			return
		}
	}

	// 解析[xxx]行
	if strings.HasPrefix(trimStr, "[") {
		strArr := findSliceString(trimStr)
		if len(strArr) > 0 {
			// 重置
			tomlLineNode = TomlLineNode{}

			tempKey := strArr[0]
			tomlLineNode.KeyName = tempKey[1 : len(tempKey)-1]

			return
		}
	}

	// 解析line
	if rxNode.MatchString(rowStr) {
		parseKv(trimStr)
	}
}

/**
 * 解析节点
 * @param rowStr string
 */
func parseKv(rowStr string) {

	posEq := strings.IndexAny(rowStr, "=")

	if posEq != -1 {

		keyName := rowStr[:posEq]

		valueStrRow := rowStr[posEq+1:]

		// 处理等号后面的值
		valueStr := parsNodeValue(valueStrRow)

		// 处理连续的分隔符
		keyName = parsNodeName(keyName)

		// 表格数组
		if tomlLineNode.Operate == "array" {
			if valueStr != "" {
				tomlLineNode.Arr[tomlLineNode.ArrIndex][keyName] = valueStr
				fmt.Println(tomlLineNode.KeyName)
				setTomlGlobalMapValue(tomlLineNode.KeyName, tomlLineNode.Arr)
				return
			}
		}

		keyName = tomlLineNode.KeyName + "." + keyName

		// 处理变量引用
		if strings.HasPrefix(valueStr, "${") {
			if parseVariate(keyName, valueStr) {
				return
			}
		}

		// 处理数组
		valueStrRow = strings.TrimSpace(valueStrRow)
		if strings.HasPrefix(valueStrRow, "[") {
			strArr := findSliceString(valueStrRow)
			if len(strArr) > 0 {
				retSlice := parseSliceRow(valueStrRow, 0)

				setTomlGlobalMapValue(keyName, retSlice)

				return
			}
		}

		setTomlGlobalMapValue(keyName, valueStr)
	}
}
