package goini

import (
	"fmt"
	"strings"
)

// 记录当前行数据
type LineNode struct {
	KeyName  string
	Data     string
	SpaceLen int64
	KeyDepth int64
	List     []map[string]interface{}
	Arr      []interface{}
	Operate  string
}

var lineNode LineNode

// 设置全局map表内容
func setGlobalMapValue(value interface{}) {
	//设置属性
	setProperty(lineNode.KeyName, value)
	//设置值
	property[lineNode.KeyName] = value
}

func parseYamlLine(rowB []byte) {
	// 去除空白，空格等字符
	rowStr := string(rowB)
	trimStr := strings.TrimSpace(rowStr)
	if trimStr == "" || trimStr == ":" {
		return
	}

	// 解析注释行
	if strings.HasPrefix(trimStr, "#") {
		return
	}

	// 数组
	if strings.Index(trimStr, "-") == 0 {
		parseArrayLine(trimStr[1:])
		return
	}

	// 重置状态
	lineNode.List = nil
	lineNode.Arr = nil

	// 如果lineNode.Operate != " < 或 | ",
	// 下一行内容如果含有:符号，肯能解析错误
	pos := strings.Index(trimStr, ":")
	if pos > 0 {
		// 第一key
		if rowB[0] != 32 {
			lineNode.KeyName = trimStr[:pos]
			lineNode.SpaceLen = 0
			lineNode.KeyDepth = 1
		} else {
			// 设置行内容
			setLineNode(trimStr[:pos], rowB)
		}

		// 重置 operate
		lineNode.Operate = ""
		lineNode.Data = ""
	}

	rowValue := strings.TrimSpace(trimStr[pos+1:])

	// 处理变量引用
	if strings.HasPrefix(rowValue, "$") {
		varStr := rxVariate.FindString(trimStr)
		if varStr != "" && len(varStr) > 3 {
			setGlobalMapValue(property[varStr[2:len(varStr)-1]])
			return
		}
	}

	// 处理数组行
	if strings.HasPrefix(rowValue, "[") {
		strArr := findSliceString(rowValue)
		if len(strArr) > 0 {
			retSlice := parseSliceRow(rowValue, 0)
			setGlobalMapValue(retSlice)
			return
		}
	}

	// 处理 flow 行
	if strings.HasPrefix(rowValue, "{") {
		flowStr := rxYamlFlow.FindString(rowValue)
		if strings.Index(flowStr, ":") != -1 {
			parseFlowRow(lineNode.KeyName, flowStr, 0)
			return
		}
	}

	if rowValue != "" {
		if rowValue == "|" || rowValue == ">" {
			lineNode.Operate = rowValue
			lineNode.Data = ""
			return
		}

		if lineNode.Operate == "|" {
			if lineNode.Data == "" {
				lineNode.Data = rowValue
			} else {
				lineNode.Data += "\n" + rowValue
			}

			setGlobalMapValue(lineNode.Data)
			return
		}

		if lineNode.Operate == ">" {
			if lineNode.Data == "" {
				lineNode.Data = rowValue
			} else {
				lineNode.Data += " " + rowValue
			}

			setGlobalMapValue(lineNode.Data)
			return
		}

		lineNode.Operate = ""
		setValStr := parsNodeValue(rowValue)
		setGlobalMapValue(setValStr)
	} else {
		lineNode.Data = ""
		lineNode.Operate = ""
	}
}

// 处理数组标识行
func parseArrayLine(trimStr string) {
	valStr := strings.TrimSpace(trimStr)
	valStr = parsNodeValue(valStr)
	if lineNode.Data == "" {
		lineNode.Data = valStr
	} else {
		lineNode.Data += "," + valStr
	}

	if strings.HasPrefix(valStr, "{") {
		retMap := parseSliceFlowRowToMap(lineNode.KeyName, valStr, 0)
		lineNode.List = append(lineNode.List, retMap)

		setGlobalMapValue(lineNode.List)

		return
	}

	// 是否是数组
	if strings.HasPrefix(valStr, "[") { // 数组内容
		retSlice := parseSliceRow(valStr, 0)
		lineNode.Arr = append(lineNode.Arr, retSlice)

		setGlobalMapValue(lineNode.Arr)

		return
	}

	setGlobalMapValue(lineNode.Data)
}

// 设置行数据
func setLineNode(rowStr string, rowB []byte) {
	// 从头计算空格长度
	spaceLen := int64(0)
	rowLen := len(rowB)
	for idx, _ := range rowB {
		if rowB[idx] == 32 {
			if idx == 0 {
				spaceLen++
			}

			nextIdx := idx + 1
			if nextIdx >= rowLen-1 {
				nextIdx = rowLen - 1
			}

			if rowB[nextIdx] == 32 {
				spaceLen++
			} else {
				break
			}
		}
	}

	tempB := []byte(lineNode.KeyName)
	periodCount := int64(0)
	periodPos := 0

	if lineNode.SpaceLen < spaceLen {
		lineNode.KeyName += "." + rowStr
		lineNode.SpaceLen = spaceLen
		lineNode.KeyDepth++
	} else if lineNode.SpaceLen >= spaceLen {
		if spaceLen == 1 {
			spaceLen = 2
		}

		num := lineNode.SpaceLen / spaceLen
		depth := lineNode.KeyDepth - num
		for i, v := range tempB {
			// 计数空格
			if v == 46 {
				periodCount++
				periodPos = i
			}

			if periodCount == depth {
				break
			}
		}

		lineNode.KeyName = string(tempB[:periodPos]) + "." + rowStr
		lineNode.KeyDepth = depth + 1
		lineNode.SpaceLen = spaceLen
	}
}

// 解析flow格式数据
func parseSliceFlowRowToMap(keyName, valStr string, depth int) map[string]interface{} {
	strLen := len(valStr)
	valStr = valStr[1 : strLen-1]

	nextFlowArr := findFlowString(valStr)
	tempMap := make(map[string]string)
	for idx, v := range nextFlowArr {
		if strings.Index(v, ":") != -1 { // 含有kv形式的进行替换，待递归处理
			destIdx := "$flow_next_" + fmt.Sprintf("%v", depth) + "_" + fmt.Sprintf("%v", idx)
			valStr = strings.Replace(valStr, v, destIdx, -1)
			tempMap[destIdx] = v
		}
	}

	ret := make(map[string]interface{})

	strArr := strings.Split(valStr, ",")
	for _, kvStr := range strArr {
		pos := strings.Index(kvStr, ":")
		if pos != -1 {
			k, v := strings.TrimSpace(kvStr[:pos]), strings.TrimSpace(kvStr[pos+1:])

			nextKey := k
			if nextV, ok := tempMap[v]; ok {
				ret[nextKey] = parseSliceFlowRowToMap(nextKey, nextV, depth+1)
				continue
			}

			ret[nextKey] = v
		}
	}

	return ret
}

// 解析flow格式数据
func parseFlowRow(keyName, valStr string, depth int) {
	strLen := len(valStr)
	valStr = valStr[1 : strLen-1]

	nextFlowArr := findFlowString(valStr)
	tempMap := make(map[string]string)
	for idx, v := range nextFlowArr {
		if strings.Index(v, ":") != -1 { // 含有kv形式的进行替换，待递归处理
			destIdx := "$flow_next_" + fmt.Sprintf("%v", depth) + "_" + fmt.Sprintf("%v", idx)
			valStr = strings.Replace(valStr, v, destIdx, -1)
			tempMap[destIdx] = v
		}
	}

	strArr := strings.Split(valStr, ",")
	for _, kvStr := range strArr {
		pos := strings.Index(kvStr, ":")
		if pos != -1 {
			k, v := strings.TrimSpace(kvStr[:pos]), strings.TrimSpace(kvStr[pos+1:])

			nextKey := keyName + "." + k
			if nextV, ok := tempMap[v]; ok {
				parseFlowRow(nextKey, nextV, depth+1)
				continue
			}

			setGlobalMapValue(v)
		}
	}
}

// 解析flow格式数据
func parseSliceRow(valStr string, depth int) []interface{} {
	strLen := len(valStr)
	valStr = valStr[1 : strLen-1]

	nextFlowArr := findSliceString(valStr)
	tempMap := make(map[string]string)
	for idx, v := range nextFlowArr {
		if strings.HasPrefix(v, "[") { // 含有kv形式的进行替换，待递归处理
			destIdx := "$slice_next_" + fmt.Sprintf("%v", depth) + "_" + fmt.Sprintf("%v", idx)
			valStr = strings.Replace(valStr, v, destIdx, -1)
			tempMap[destIdx] = v
		}
	}

	var retSlice []interface{}

	strArr := strings.Split(valStr, ",")
	for _, valStr := range strArr {
		valStr = strings.TrimSpace(valStr)
		if val, ok := tempMap[valStr]; ok {
			nextVal := parseSliceRow(val, depth+1)
			retSlice = append(retSlice, nextVal)
			continue
		}

		retSlice = append(retSlice, valStr)
	}

	return retSlice
}

// 找出匹配flow格式的字符串，只匹配第一层
func findFlowString(str string) []string {
	return findClosureValue(str, 123, 125, "}", "{")
}

// 找出数组格式数据
func findSliceString(str string) []string {
	return findClosureValue(str, 91, 93, "]", "[")
}

func findClosureValue(str string, start, end byte, endSymbol, startSymbol string) []string {
	strB := []byte(str)

	var tempB []byte
	var retArr []string

	// 哨兵
	guard := 0

	for _, v := range strB {
		if v == start {
			guard++ // 放置哨兵
			tempB = append(tempB, v)
			continue
		}

		if v == end {
			guard-- // 清除哨兵
			tempB = append(tempB, v)
			if guard == 0 {
				retArr = append(retArr, string(tempB))
				//重置空
				tempB = []byte{}
			}
			continue
		}

		if guard > 0 {
			tempB = append(tempB, v)
		}
	}

	if guard > 0 {
		panic(" '" + endSymbol + "' Symbol mismatch")
	}

	if guard < 0 {
		panic(" '" + startSymbol + "' Symbol mismatch")
	}

	// 回收临时变量
	tempB = []byte{}
	strB = []byte{}

	return retArr
}
