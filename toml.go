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
	IsChild   int
	Data      string
	Arr       []map[string]interface{}
	ArrDepth  string
	Operate   string
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
			tomlLineNode.Operate = OpArrayTable

			// 处理数组索引为平滑结构 arr[0][1][2]...[n] => 0.1.2...n
			JoinIndex(keyName)

			// 设置数组表格
			// setArrayTable(keyName, "", "")

			return
		}
	}

	// 解析[xxx]行
	if strings.HasPrefix(trimStr, "[") && strings.HasSuffix(trimStr, "]") {
		parseTomlArrayLine()

		strArr := findSliceString(trimStr)
		if len(strArr) > 0 {
			tempKey := strArr[0]
			keyName := tempKey[1 : len(tempKey)-1]

			if tomlLineNode.Operate == OpArrayTable || tomlLineNode.Operate == OpArrayTableChild {
				tomlLineNode.KeyName = keyName
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

	// 解析line
	if rxNode.MatchString(rowStr) {
		parseTomlArrayLine()

		parseKv(trimStr)
	} else {
		if trimStr == "\"\"\"" || trimStr == "'''" {
			stringSetAndReset()
			return
		} else {
			if tomlLineNode.Operate == OpInLine {
				if strings.HasSuffix(trimStr, "\"\"\"") || strings.HasSuffix(trimStr, "'''") {
					tomlLineNode.Data += trimStr[:len(trimStr)-3]

					stringSetAndReset()

					return
				} else {
					trimStr = strings.TrimRight(trimStr, "\\")
					tomlLineNode.Data += trimStr
				}
			}

			if tomlLineNode.Operate == OpNewLine {
				if strings.HasSuffix(trimStr, "\"\"\"") || strings.HasSuffix(trimStr, "'''") {
					tomlLineNode.Data += trimStr[:len(trimStr)-3]

					stringSetAndReset()

					return
				} else {
					// 保持原格式
					tomlLineNode.Data += rowStr + "\n"
				}
			}

			if tomlLineNode.Operate == OpArrayLine {
				tomlLineNode.Data += trimStr
			}
		}
	}
}

// 设置值，并重置
func stringSetAndReset() {
	setTomlGlobalMapValue(tomlLineNode.KeyName, tomlLineNode.Data)
	tomlLineNode.Data = ""
	tomlLineNode.KeyName = tomlLineNode.ParentKey
	tomlLineNode.Operate = ""
}

func parseTomlArrayLine() {
	if tomlLineNode.Operate == OpArrayLine {
		strArr := findSliceString(tomlLineNode.Data)

		if len(strArr) > 0 {
			retSlice := parseSliceRow(tomlLineNode.Data, 0)
			setTomlGlobalMapValue(tomlLineNode.KeyName, retSlice)

			return
		}

		// 重置
		tomlLineNode.Data = ""
		tomlLineNode.KeyName = ""
		tomlLineNode.Operate = ""
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
		// 处理 """ 行
		trimStr := strings.TrimSpace(valueStrRow)

		// string 单行
		if (strings.HasPrefix(trimStr, "\"\"\"") && strings.HasSuffix(trimStr, "\"\"\"")) ||
			(strings.HasPrefix(trimStr, "'''") && strings.HasSuffix(trimStr, "'''")) {

			strLen := len(trimStr)
			if strLen > 6 {
				keyName = tomlLineNode.ParentKey + "." + keyName
				setTomlGlobalMapValue(keyName, trimStr[3:len(trimStr)-3])
				tomlLineNode.Operate = ""
				return
			}
		}

		// string 多行
		if strings.HasPrefix(trimStr, "\"\"\"") || strings.HasPrefix(trimStr, "'''") {
			tomlLineNode.KeyName = tomlLineNode.ParentKey + "." + keyName
			if len(trimStr) > 3 {
				tomlLineNode.Operate = OpInLine
			} else {
				tomlLineNode.Operate = OpNewLine
			}

			return
		}

		// 处理数组换行
		valueStr := parsNodeValue(valueStrRow)

		if valueStr == "[" {
			tomlLineNode.KeyName = tomlLineNode.ParentKey + "." + keyName
			tomlLineNode.Operate = OpArrayLine
			tomlLineNode.Data = valueStr
			return
		}

		// 表格数组
		if tomlLineNode.Operate == OpArrayTable || tomlLineNode.Operate == OpArrayTableChild {
			// 设置数组表格
			setArrayTable(tomlLineNode.KeyName, keyName, valueStr)

			return
		}

		newKeyName := tomlLineNode.KeyName + "." + keyName

		// 处理内联表
		if strings.HasPrefix(trimStr, "{") {
			flowStr := rxYamlFlow.FindString(trimStr)

			if strings.Index(flowStr, "=") != -1 {
				parseInlineRow(newKeyName, flowStr, 0)
				return
			}
		}

		// 处理变量引用
		if strings.HasPrefix(valueStr, "${") {
			if parseVariate(newKeyName, valueStr) {
				return
			}
		}

		// 处理数组
		valueStrRow = strings.TrimSpace(valueStrRow)
		if strings.HasPrefix(valueStrRow, "[") {
			strArr := findSliceString(valueStrRow)
			if len(strArr) > 0 {
				retSlice := parseSliceRow(valueStrRow, 0)

				setTomlGlobalMapValue(newKeyName, retSlice)

				return
			}
		}

		setTomlGlobalMapValue(newKeyName, valueStr)
	}
}

// 解析嵌套数组表格
func parseArrayTable(rootKey, currentKey string, valueStr string, keyDepth, depth int, obj interface{}, keyArr []string) interface{} {
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
			//fmt.Println(preKey, rootKey, keyDepth, depth)
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
func setArrayTable(keyName, valueKey, valueStr string) {
	keyArr := strings.Split(keyName, ".")
	obj := parseArrayTable(keyName, valueKey, valueStr, len(keyArr)-1, 0, nil, keyArr)

	// 解析时增加了一层处理，这里需要还原回来
	if mp, ok := obj.(map[string]interface{}); ok {
		obj = mp[keyArr[0]]
	}

	setTomlGlobalMapValue(keyArr[0], obj)
}

// 解析flow格式数据
func parseInlineRow(keyName, valStr string, depth int) {
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

	strArr := strings.Split(valStr, ",")

	for _, kvStr := range strArr {
		pos := strings.Index(kvStr, "=")
		if pos != -1 {
			k, v := strings.TrimSpace(kvStr[:pos]), strings.TrimSpace(kvStr[pos+1:])

			nextKey := keyName + "." + k
			if nextV, ok := tempMap[v]; ok {
				parseInlineRow(nextKey, nextV, depth+1)

				continue
			}

			v = TrimQuote(v)

			setTomlGlobalMapValue(keyName+"."+k, v)
		}
	}
}

// 去除首尾引号
func TrimQuote(dest string) string {
	destB := []byte(dest)
	strLen := len(destB)

	if strLen < 2 {
		return dest
	}

	if destB[0] == 34 && destB[strLen-1] == 34 {
		destB = destB[1 : strLen-1]
		return string(destB)
	}

	if destB[0] == 39 && destB[strLen-1] == 39 {
		destB = destB[1 : strLen-1]
		return string(destB)
	}

	return string(destB)
}

// 去除key后面的注释
func RemoveComments(dest string) string {
	pos := strings.IndexAny(dest, "#")
	if pos > 0 {
		return strings.TrimSpace(dest[:pos])
	}

	return dest
}
