package goini

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	OpInLine          = "in-line"
	OpNewLine         = "new-line"
	OpArrayTable      = "array-table"
	OpArrayTableChild = "table_child"
	OpArrayLine       = "array-line"
)

// 记录当前行数据
type TomlLineNode struct {
	KeyName   string
	ParentKey string
	Data      string
	ArrDepth  string
	Operate   string
	MultiLine string
}

var tomlLineNode = TomlLineNode{
	KeyName: defaultName,
}

var IdxMap = map[string]string{}

// 设置全局map表内容
func setTomlGlobalMapValue(keyName string, value interface{}) {
	//设置属性
	setProperty(keyName, value)
	//设置值
	//property[keyName] = value
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

	if strings.HasPrefix(trimStr, "[") {
		trimStr = RemoveComments(trimStr)
	}

	// 解析[[xxx]]
	if strings.HasPrefix(trimStr, "[[") && strings.HasSuffix(trimStr, "]]") {
		parseTomlArrayLine()

		strArr := findSliceString(trimStr)
		if len(strArr) > 0 {
			tempKey := strArr[0]
			keyName := tempKey[2 : len(tempKey)-2]

			tomlLineNode.KeyName = keyName
			tomlLineNode.ParentKey = keyName
			tomlLineNode.Operate = OpArrayTable

			// 处理数组索引为平滑结构 arr[0][1][2]...[n] => 0.1.2...n
			JoinIndex(keyName)

			// 设置数组表格
			// setArrayTable(keyName, "", "")

			return
		}
	}

	// 解析[xxx]行
	if strings.HasPrefix(trimStr, "[") && strings.HasSuffix(trimStr, "]") && tomlLineNode.Operate != OpInLine {
		strArr := findSliceString(trimStr)
		if len(strArr) > 0 {
			tempKey := strArr[0]
			keyName := tempKey[1 : len(tempKey)-1]

			if tomlLineNode.Operate == OpArrayTable || tomlLineNode.Operate == OpArrayTableChild {
				tomlLineNode.KeyName = keyName
				tomlLineNode.ParentKey = keyName
				tomlLineNode.Operate = OpArrayTableChild

				return
			}

			// 重置
			tomlLineNode = TomlLineNode{}

			tomlLineNode.KeyName = keyName
			tomlLineNode.ParentKey = keyName

			return
		}
	}

	// 拼装多行为一行数据
	if tomlLineNode.MultiLine == OpInLine || tomlLineNode.MultiLine == OpNewLine {
		if joinMultiLine(trimStr) {
			pos := strings.LastIndexAny(tomlLineNode.KeyName, ".")
			keyName := tomlLineNode.KeyName
			if pos > -1 {
				keyName = tomlLineNode.KeyName[pos+1:]
			}

			tomlLineNode.KeyName = tomlLineNode.ParentKey
			trimStr = keyName + "=" + tomlLineNode.Data
		} else {
			return
		}
	}

	if rxNode.MatchString(trimStr) {
		parseKv(trimStr)
	}
}

// 解析多行数据
func joinMultiLine(rowStr string) bool {
	trimStr := strings.TrimSpace(rowStr)
	trimStrComment := RemoveComments(trimStr)
	trimStrQuote := TrimQuote(trimStrComment)

	if tomlLineNode.MultiLine == OpInLine || tomlLineNode.MultiLine == OpNewLine {
		if strings.HasSuffix(trimStrComment, "\"\"\"") || strings.HasSuffix(trimStrComment, "'''") {
			tomlLineNode.Data += trimStrComment[:len(trimStrComment)-3]

			return true
		} else if trimStrQuote == "]" {
			tomlLineNode.Data += trimStrQuote
			tomlLineNode.MultiLine = OpArrayLine

			return true
		} else {
			if strings.HasSuffix(trimStrQuote, "\\") || strings.HasPrefix(tomlLineNode.Data, "[") {
				trimStr = strings.TrimRight(trimStrQuote, "\\")
				tomlLineNode.Data += trimStr
			} else {
				tomlLineNode.Data += trimStrQuote + "\n"
				tomlLineNode.MultiLine = OpNewLine
			}
		}
	}

	return false
}

// 设置值，并重置
func stringSetAndReset() {
	tomlLineNode.Data = ""
	tomlLineNode.KeyName = tomlLineNode.ParentKey
	tomlLineNode.MultiLine = ""
}

func parseTomlArrayLine() {
	if tomlLineNode.MultiLine == OpArrayLine {
		strArr := findSliceString(tomlLineNode.Data)
		if len(strArr) > 0 {
			retSlice := parseInlineSliceRow(tomlLineNode.KeyName, tomlLineNode.Data, 0)
			setTomlGlobalMapValue(tomlLineNode.KeyName, retSlice)
		}

		// 重置
		tomlLineNode.Data = ""
		tomlLineNode.KeyName = tomlLineNode.ParentKey
		tomlLineNode.MultiLine = ""
	}
}

/**
 * 解析key = value 格式
 * @param rowStr string
 */
func parseKv(rowStr string) {
	posEq := strings.IndexAny(rowStr, "=")

	if posEq != -1 {

		keyName := rowStr[:posEq]
		// 处理连续的分隔符
		keyName = parsNodeName(keyName)

		valueStrRow := rowStr[posEq+1:]
		trimStr := strings.TrimSpace(valueStrRow)
		trimStrComment := RemoveComments(trimStr)
		trimStrQuote := TrimQuote(trimStrComment)

		newKeyName := tomlLineNode.KeyName + "." + keyName

		if tomlLineNode.MultiLine == OpInLine || tomlLineNode.MultiLine == OpNewLine || tomlLineNode.MultiLine == OpArrayLine {
			if tomlLineNode.Operate == "" {
				if tomlLineNode.MultiLine == OpArrayLine {
					if v, ok := parseSliceValue(keyName, trimStrComment); ok {
						setTomlGlobalMapValue(newKeyName, v)
					}
				} else {
					setTomlGlobalMapValue(newKeyName, trimStrComment)
				}
			} else {
				var rowVal interface{}
				// 处理数组
				if v, ok := parseSliceValue(newKeyName, trimStr); ok {
					rowVal = v
				}

				if rowVal == nil {
					rowVal = trimStrQuote
				}

				setArrayTable(tomlLineNode.KeyName, keyName, rowVal)
			}

			stringSetAndReset()
			return
		}

		// string 单行
		if (strings.HasPrefix(trimStrComment, "\"\"\"") && strings.HasSuffix(trimStrComment, "\"\"\"")) ||
			(strings.HasPrefix(trimStrComment, "'''") && strings.HasSuffix(trimStrComment, "'''")) {

			strLen := len(trimStrComment)
			if strLen > 6 {
				keyName = tomlLineNode.ParentKey + "." + keyName
				setTomlGlobalMapValue(keyName, trimStrComment[3:len(trimStrComment)-3])
				tomlLineNode.Operate = ""
				tomlLineNode.Data = ""
				return
			}
		}

		// string 多行
		if strings.HasPrefix(trimStrComment, "\"\"\"") || strings.HasPrefix(trimStrComment, "'''") {
			tomlLineNode.KeyName = tomlLineNode.ParentKey + "." + keyName
			if len(trimStrComment) >= 3 {
				tomlLineNode.MultiLine = OpInLine
			}

			return
		}

		// 处理数组换行
		if strings.HasPrefix(trimStrComment, "[") && !strings.HasSuffix(trimStrComment, "]") {
			tomlLineNode.KeyName = tomlLineNode.ParentKey + "." + keyName
			tomlLineNode.MultiLine = OpInLine
			tomlLineNode.Data = trimStrQuote
			return
		}

		// 表格数组
		if tomlLineNode.Operate == OpArrayTable || tomlLineNode.Operate == OpArrayTableChild {
			// 处理内联表
			var rowVal interface{}
			if v, ok := parseMapValue(keyName, trimStr); ok {
				rowVal = v
			}

			// 处理数组
			if v, ok := parseSliceValue(newKeyName, trimStr); ok {
				rowVal = v
			}

			if rowVal == nil {
				rowVal = trimStrQuote
			}

			setArrayTable(tomlLineNode.KeyName, keyName, rowVal)
			return
		}

		// 处理内联表
		if v, ok := parseMapValue(tomlLineNode.KeyName, trimStrComment); ok {
			setTomlGlobalMapValue(newKeyName, v)
			return
		}

		// 处理变量引用
		if strings.HasPrefix(trimStrQuote, "${") {
			if parseVariate(newKeyName, trimStrQuote) {
				return
			}
		}

		// 处理数组
		if v, ok := parseSliceValue(newKeyName, trimStrComment); ok {
			setTomlGlobalMapValue(newKeyName, v)
			return
		}

		// 无特殊情况数值
		if tomlLineNode.KeyName == defaultName {
			newKeyName = keyName
		}

		setTomlGlobalMapValue(newKeyName, trimStrQuote)
	}
}

// 解析素组类型的值
func parseSliceValue(keyName, lineValue string) (interface{}, bool) {
	if strings.HasPrefix(lineValue, "[") {
		strArr := findSliceString(lineValue)
		if len(strArr) > 0 {
			return parseInlineSliceRow(keyName, lineValue, 0), true
		}
	}

	return nil, false
}

// 解析内联表类型的值
func parseMapValue(keyName, lineValue string) (interface{}, bool) {
	if strings.HasPrefix(lineValue, "{") && strings.HasSuffix(lineValue, "}") {
		if strings.Index(lineValue, "=") != -1 {
			return parseInlineRow(keyName, lineValue, 0, nil), true
		}
	}

	return nil, false
}

// 解析嵌套数组表格
func parseArrayTable(rootKey, currentKey string, valueStr interface{}, keyDepth, depth int, obj interface{}, keyArr []string) interface{} {
	if obj == nil {
		if depth == 0 {
			tempRootKey := keyArr[depth]
			obj = getValBySection(tempRootKey, "")
			if obj == nil {
				obj = make(map[string]interface{})
				// 如果根值是数组，则进行数组初始化
				if tomlLineNode.Operate == OpArrayTable || tomlLineNode.Operate == OpArrayTableChild {
					newArr := make([]map[string]interface{}, 0)
					newMap := make(map[string]interface{})
					newArr = append(newArr, newMap)
					if mp, ok := obj.(map[string]interface{}); ok {
						mp[tempRootKey] = newArr
					}
				} else {
					if mp, ok := obj.(map[string]interface{}); ok {
						mp[tempRootKey] = make(map[string]interface{})
					}
				}
			} else {
				// 若果有值，则增加一层map映射
				if mp, ok := obj.(map[string]interface{}); ok {
					if _, ok := mp[tempRootKey]; !ok {
						mp := make(map[string]interface{})
						mp[tempRootKey] = obj

						obj = mp
					}
				} else if _, ok := obj.([]map[string]interface{}); ok {
					mp := make(map[string]interface{})
					mp[tempRootKey] = obj

					obj = mp
				}
			}
		}
	}

	// 如果达到同等深度, 则开始进行赋值
	if depth == keyDepth && keyDepth > 0 {
		if mp, ok := obj.(map[string]interface{}); ok {
			v := mp[rootKey]
			// 若当前key的map没有值，则进行初始化
			if v == nil {
				if tomlLineNode.Operate == OpArrayTable {
					newArr := make([]map[string]interface{}, 0)
					newMap := make(map[string]interface{})

					if currentKey != "" {
						newMap[currentKey] = valueStr
					}

					newArr = append(newArr, newMap)

					mp[rootKey] = newArr
				} else {
					newMap := make(map[string]interface{})

					if currentKey != "" {
						newMap[currentKey] = valueStr
					}

					mp[rootKey] = newMap
				}
			} else {
				// 若有值，并且是数组类型，则根据索引进行初始化
				if tomlLineNode.Operate == OpArrayTable {
					if arr, ok := v.([]map[string]interface{}); ok {
						arrLen := len(arr)

						strIdxArr := strings.Split(tomlLineNode.ArrDepth, ".")
						strIdx := strIdxArr[depth]
						iIdx, _ := strconv.Atoi(strIdx)

						if arrLen < iIdx {
							for i := arrLen; i < iIdx; i++ {
								newMp := make(map[string]interface{})
								if i == iIdx-1 && currentKey != "" {
									newMp[currentKey] = valueStr
								}

								arr = append(arr, newMp)
							}
						} else {
							if currentKey != "" {
								m := arr[arrLen-1]
								m[currentKey] = valueStr
							}
						}

						mp[rootKey] = arr
					}
				} else {
					if val, ok := v.(map[string]interface{}); ok && currentKey != "" {
						val[currentKey] = valueStr
					}
				}
			}
		}
	} else {
		childArr := keyArr[:depth+1]
		preKey := strings.Join(childArr, ".")
		var nextObj interface{}
		if mp, ok := obj.(map[string]interface{}); ok {
			v := mp[preKey]
			if v == nil {
				v = make(map[string]interface{})
				mp[preKey] = v

				nextObj = v
			} else {
				if arr, ok := v.([]map[string]interface{}); ok {
					strIdxArr := strings.Split(tomlLineNode.ArrDepth, ".")
					strIdx := strIdxArr[depth]
					iIdx, _ := strconv.Atoi(strIdx)

					arrLen := len(arr)
					if arrLen < iIdx {
						for i := arrLen; i < iIdx; i++ {
							newMp := make(map[string]interface{})

							if i == iIdx-1 {
								if keyDepth == 0 && currentKey != "" {
									newMp[currentKey] = valueStr
								}

								nextObj = newMp
							}

							arr = append(arr, newMp)
						}
					} else {
						nextObj = arr[arrLen-1]
						if keyDepth == 0 {
							if m, ok := nextObj.(map[string]interface{}); ok && currentKey != "" {
								m[currentKey] = valueStr
							}
						}
					}

					mp[preKey] = arr
				}

				if mp, ok := v.(map[string]interface{}); ok {
					if keyDepth == 0 && currentKey != "" {
						mp[currentKey] = valueStr
					}

					nextObj = mp
				}
			}
		}

		if keyDepth == 0 {
			return obj
		}

		// 递归解析
		parseArrayTable(rootKey, currentKey, valueStr, keyDepth, depth+1, nextObj, keyArr)
	}

	return obj
}

func CountChar(destStr, charStr string) int {
	strB := []byte(destStr)
	charArr := []byte(charStr)
	char := charArr[0]

	num := 0
	for _, v := range strB {
		if char == v {
			num++
		}
	}

	return num
}

func GetRootKey(str string) string {
	pos := strings.IndexAny(str, ".")

	rootKey := str
	if pos > 0 {
		rootKey = str[:pos]
	}

	return rootKey
}

// 多维数组索引平滑处理 arr[0][1][2] => 0.1.2.3.4
func JoinIndex(keyName string) {
	rootKey := GetRootKey(keyName)
	if idxStr, ok := IdxMap[rootKey]; ok {
		tomlLineNode.ArrDepth = idxStr
	} else {
		IdxMap[rootKey] = ""
		tomlLineNode.ArrDepth = ""
	}

	num := CountChar(keyName, ".")
	if tomlLineNode.ArrDepth == "" {
		tomlLineNode.ArrDepth = "0"
	}

	strIdxArr := strings.Split(tomlLineNode.ArrDepth, ".")
	strIdxArrLen := len(strIdxArr)
	if strIdxArrLen <= num {
		for i := strIdxArrLen; i <= num; i++ {
			strIdxArr = append(strIdxArr, "0")
		}
	} else {
		for i := num + 1; i < strIdxArrLen; i++ {
			strIdxArr[i] = "0"
		}
	}

	strIdx := strIdxArr[num]
	iIdx, _ := strconv.Atoi(strIdx)
	iIdx++
	strIdxArr[num] = strconv.Itoa(iIdx)

	tomlLineNode.ArrDepth = strings.Join(strIdxArr, ".")

	IdxMap[rootKey] = tomlLineNode.ArrDepth
}

// 设置数组表格
func setArrayTable(keyName, valueKey string, valueStr interface{}) {
	keyArr := strings.Split(keyName, ".")
	obj := parseArrayTable(keyName, valueKey, valueStr, len(keyArr)-1, 0, nil, keyArr)

	// 解析时增加了一层处理，这里需要还原回来
	if mp, ok := obj.(map[string]interface{}); ok {
		obj = mp[keyArr[0]]
	}

	setTomlGlobalMapValue(keyArr[0], obj)
}

// 解析flow格式数据
func parseInlineRow(keyName, valStr string, depth int, mp map[string]interface{}) interface{} {
	if mp == nil {
		mp = make(map[string]interface{})
	}

	strLen := len(valStr)
	valStr = valStr[1 : strLen-1]

	nextFlowArr := findFlowString(valStr)
	tempMap := make(map[string]string)
	for idx, v := range nextFlowArr {
		if strings.Index(v, "=") != -1 { // 含有kv形式的进行替换，待递归处理
			destIdx := "$flow_next_" + fmt.Sprintf("%v", depth) + "_" + fmt.Sprintf("%v", idx)
			valStr = strings.Replace(valStr, v, destIdx, -1)
			tempMap[destIdx] = v
		}
	}

	oldMp := mp[keyName]
	if oldMp == nil {
		oldMp = make(map[string]interface{})
		mp[keyName] = oldMp
	}

	// 如果含有数组
	sliceArr := findSliceString(valStr)
	tempSliceMap := make(map[string]string)
	if len(sliceArr) > 0 {
		for idx, v := range sliceArr {
			destIdx := "$slice" + "_" + fmt.Sprintf("%v", idx)
			valStr = strings.Replace(valStr, v, destIdx, -1)
			tempSliceMap[destIdx] = v
		}
	}

	strArr := strings.Split(valStr, ",")
	if m, ok := oldMp.(map[string]interface{}); ok {
		for _, kvStr := range strArr {
			pos := strings.Index(kvStr, "=")
			if pos != -1 {
				k, v := strings.TrimSpace(kvStr[:pos]), strings.TrimSpace(kvStr[pos+1:])

				v = TrimQuote(v)
				//nextKey := keyName + "." + k
				if nextV, ok := tempMap[v]; ok {
					parseInlineRow(k, nextV, depth+1, m)
					continue
				}

				if sliceStr, ok := tempSliceMap[v]; ok {
					retSlice := parseInlineSliceRow(k, sliceStr, 0)
					m[k] = retSlice

					continue
				}

				m[k] = v
			}
		}
	}

	return mp[keyName]
}

// 解析flow格式数据
func parseInlineSliceRow(keyName, valStr string, depth int) []interface{} {
	var retSlice []interface{}

	strLen := len(valStr)
	valStr = valStr[1 : strLen-1]

	// 如果含有内联表
	flowArr := findFlowString(valStr)
	if len(flowArr) > 0 {
		for _, flowStr := range flowArr {
			retMap := parseInlineRow(keyName, flowStr, 0, nil)
			retSlice = append(retSlice, retMap)
		}

		return retSlice
	}

	nextFlowArr := findSliceString(valStr)
	tempMap := make(map[string]string)
	for idx, v := range nextFlowArr {
		if strings.HasPrefix(v, "[") { // 含有kv形式的进行替换，待递归处理
			destIdx := "$slice_next_" + fmt.Sprintf("%v", depth) + "_" + fmt.Sprintf("%v", idx)
			valStr = strings.Replace(valStr, v, destIdx, -1)
			tempMap[destIdx] = v
		}
	}

	valStr = strings.TrimRight(valStr, ",")
	strArr := strings.Split(valStr, ",")
	for _, valStr := range strArr {
		valStr = strings.TrimSpace(valStr)
		valStr = TrimQuote(valStr)

		if val, ok := tempMap[valStr]; ok {
			nextVal := parseInlineSliceRow(keyName, val, depth+1)
			retSlice = append(retSlice, nextVal)
			continue
		}

		retSlice = append(retSlice, valStr)
	}

	return retSlice
}

// 去除首尾引号
func TrimQuote(dest string) string {
	destB := []byte(dest)
	strLen := len(destB)

	if strLen < 2 {
		return dest
	}

	// 去掉 " 引号
	if destB[0] == 34 && destB[strLen-1] == 34 {
		destB = destB[1 : strLen-1]
		return string(destB)
	}

	// 去掉 ' 引号
	if destB[0] == 39 && destB[strLen-1] == 39 {
		destB = destB[1 : strLen-1]
		return string(destB)
	}

	return string(destB)
}

// 获取引号符号情况下的注释开始位置
func GetQuoteCommentIndex(destB []byte, target, char byte) int {
	num := 0
	pos := -1
	for idx, v := range destB {
		if v == char {
			num++
		}

		if v == target {
			pos = idx

			if num&1 == 0 {
				break
			}
		}
	}

	return pos
}

// 获取闭合符号情况下的注释开始位置
func GetClosedCommentIndex(destB []byte, target, start, end byte) int {
	pos := -1

	// 哨兵
	guard := 0
	for idx, v := range destB {
		if v == start {
			guard++ // 放置哨兵
		}

		if v == end {
			guard-- // 清除哨兵
		}

		if v == target {
			pos = idx
			if guard == 0 {
				break
			}
		}
	}

	return pos
}

// 去除注释
func RemoveComments(dest string) string {
	if dest == "" {
		return ""
	}

	pos := -1

	destB := []byte(dest)
	switch destB[0] {
	case 34:
		pos = GetQuoteCommentIndex(destB, 35, 34)
	case 39:
		pos = GetQuoteCommentIndex(destB, 35, 39)
	case 91:
		pos = GetClosedCommentIndex(destB, 35, 91, 93)
	case 123:
		pos = GetClosedCommentIndex(destB, 35, 123, 125)
	default:
		pos = GetQuoteCommentIndex(destB, 35, 34)
	}

	if pos > -1 {
		return strings.TrimSpace(dest[:pos])
	}

	return dest
}
