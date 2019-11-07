package goini

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type Config interface {
	// 获取值，默认取default节下的节点取值
	Get(key string, args ...interface{}) interface{}

	// 设置值，默认从default节下节点设置
	Set(key string, val interface{}, args ...interface{})

	// 使用节，获取相关节点的值
	getValBySection(key string, section string) interface{}

	// 使用节，设置相关节点的值
	setValBySection(key string, val interface{}, section string)

	// 取节值
	GetSection(section string) map[string]interface{}

	// 返回string类型的值
	GetString(key string, args ...interface{}) string

	// 返回int64类型的值
	GetInt(key string, args ...interface{}) int64

	// 返回float64类型的值
	GetFloat(key string, args ...interface{}) float64

	// 返回bool类型的值
	GetBool(key string, args ...interface{}) bool

	// 转换为指定的切片
	GetSlice(key string, delimiter string, targetObj interface{}, args ...interface{})

	// 转化为map类型，obj引用传值
	GetMap(key string, targetObj interface{}, args ...interface{})

	// 转化为结构体类型，obj引用传值
	GetStruct(key string, targetObj interface{}, args ...interface{})
}

type Goini struct {
	filePath string
	Syntax   string
}

// 默认ini文件
var defaultIni string = "application.ini"

// 默认参数 -c
var iniPath = flag.String("c", "", "ini file path")

// 默认参数 -conf
var confPath = flag.String("conf", "", "ini file path")

// 存储节内容
var sections map[string]interface{}

// 存储节点内容
var property map[string]interface{}

// 存储父节内容
var parentMap map[string]interface{}

// 默认节名
const defaultName = "default"

// 记录当前节名
var sectionName string

var syntaxMap = map[string]string{
	"yaml": "yml",
	"yml":  "yml",
	"YAML": "yml",
	"YML":  "yml",

	"ini":  "ini",
	"conf": "ini",
	"":     "ini",
}

/**
 * 默认节点取值
 * @param key string 节点名称
 * @return interface{}
 */
func (goini *Goini) Get(key string, args ...interface{}) interface{} {
	var retVal interface{}

	argLen := len(args)
	if argLen > 0 {
		// 若可变参数长度大于0， 则取第一个参数为节名
		retVal = goini.getValBySection(key, fmt.Sprintf("%v", args[0]))
	} else {
		// 取默认节
		retVal = goini.getValBySection(key, defaultName)
	}

	// 可变参数长度大于2，如果未获取到值的情况下，则取第二个可变参数为默认值返回
	if argLen >= 2 {
		if retVal == nil || retVal == "" {
			retVal = args[1]
		}
	}

	return retVal
}

/**
 * 设置默认节点值
 * @param key string 节点名称
 * @param val interface{} 混合类型
 * @param args 可变参数，当长度大于0，则设置多个节
 */
func (goini *Goini) Set(key string, val interface{}, args ...interface{}) {
	if len(args) > 0 {
		for _, arg := range args {
			goini.setValBySection(key, fmt.Sprintf("%v", val), fmt.Sprintf("%v", arg))
		}
	} else {
		goini.setValBySection(key, fmt.Sprintf("%v", val), defaultName)
	}

}

/**
 * 获取指定节内容
 * @param section string 节名
 * @return map[string]interface{}
 */
func (goini *Goini) GetSection(section string) map[string]interface{} {
	return GetSection(section)
}

/**
 * 根据节和节点名称获取值
 * @param key string 节点名
 * @param section string 节名
 * @return interface{} 混合类型内容
 */
func (goini *Goini) getValBySection(key string, section string) interface{} {
	return getValBySection(key, section)
}

/**
 *  根据节和节点名设置值
 * @param key string 节点名
 * @param val string 值
 * @param section string 节名
 */
func (goini *Goini) setValBySection(key string, val interface{}, section string) {
	if section == "" {
		section = defaultName
	}

	if key == "" {
		panic("set node error: key cannot be nil")
	}

	//设置节点
	parseSection(section)

	// 设置属性
	parseProperty(key + "=" + fmt.Sprintf("%v", val))
}

// 返回string类型的值
func (goini *Goini) GetString(key string, args ...interface{}) string {
	val := goini.Get(key, args...)

	if valStr, ok := val.(string); ok {

		valStr = decodeVariable(valStr)

		return valStr
	}

	return ""
}

// 返回int64类型的值
func (goini *Goini) GetInt(key string, args ...interface{}) int64 {
	val := goini.Get(key, args...)

	if valStr, ok := val.(string); ok {

		valStr = decodeVariable(valStr)

		if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
			return int64(floatVal)
		}
	}

	return 0
}

// 返回float64类型的值
func (goini *Goini) GetFloat(key string, args ...interface{}) float64 {
	val := goini.Get(key, args...)

	if valStr, ok := val.(string); ok {

		valStr = decodeVariable(valStr)

		if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
			return floatVal
		}
	}

	return 0
}

// 返回bool类型的值
func (goini *Goini) GetBool(key string, args ...interface{}) bool {
	val := goini.Get(key, args...)

	ret, _ := parseBool(val)

	return ret
}

// 转换为切片类型
func (goini *Goini) GetSlice(key string, delimiter string, targetObj interface{}, args ...interface{}) {
	val := goini.Get(key, args...)

	objV := reflect.ValueOf(targetObj)
	objT := reflect.TypeOf(targetObj)

	// 目标对象必须是指针类型
	if objT.Kind() == reflect.Ptr {
		objV = objV.Elem()
		objT = objT.Elem()
	} else {
		return
	}

	// 目标对象需要是切片类型
	if objT.Kind() != reflect.Slice {
		return
	}

	if val == "" {
		return
	}

	if retVal, err := parseSlice(val, objT, delimiter); err == nil {
		objV.Set(retVal)
	}
}

// 转化为map类型，obj引用传值
func (goini *Goini) GetMap(key string, targetObj interface{}, args ...interface{}) {
	val := goini.Get(key, args...)

	objV := reflect.ValueOf(targetObj)
	objT := reflect.TypeOf(targetObj)

	if objT.Kind() == reflect.Ptr {
		objV = objV.Elem()
		objT = objT.Elem()
	}

	if objT.Kind() != reflect.Map {
		return
	}

	if objT.Key().Kind() != reflect.String {
		return
	}

	if retVal, err := parseMap(val, objT); err == nil {
		objV.Set(retVal)
	}
}

// 转化为结构体类型，obj引用传值
func (goini *Goini) GetStruct(key string, targetObj interface{}, args ...interface{}) {
	val := goini.Get(key, args...)

	if valMap, ok := val.(map[string]interface{}); ok {
		mapToStruct(key, valMap, targetObj)
	}
}

/**
 * 构造对象
 * @return *Goini
 */
func New(syntax string) *Goini {
	var path string

	// 命令行 获取文件
	path = ArgConfigPath()

	if path == "" {
		// 执行目录当前查找
		currentPath, err := GetCurrentPath()
		if err != nil {
			panic("goini error: ini file cannot be nil")
		}

		path = currentPath + defaultIni
	}

	return Load(path, syntax)

}

func InitFlag(flagArg, defaultFile string) {
	confPath = flag.String(flagArg, defaultFile, "conf path")
}

/**
 * 加载指定文件
 * @param path string 文件路径
 * @return *Goini
 */
func Load(path, syntax string) *Goini {
	// 文件是否存在
	isFile, _ := PathExists(path)
	if !isFile {
		panic("goini error: " + path + " not exists")
	}

	config := &Goini{
		filePath: path,
	}

	// 初始化节
	sections = make(map[string]interface{})

	// 初始化节点属性
	property = make(map[string]interface{})

	sectionName = defaultName

	sections[sectionName] = property

	// 解析文件
	err := parseFile(config.filePath, syntax)
	if err != nil {
		panic("goini error: file parse error \r\n err:" + err.Error())
	}

	return config
}

/**
 * 获取命令行中的文件路径
 * @return string
 */
func ArgConfigPath() string {
	if !flag.Parsed() {
		flag.Parse()
	}

	path := *iniPath
	if path != "" {
		return path
	}

	path = *confPath

	return path
}

/**
 * 文件是否真实存在
 * @param path string 文件路径
 * @return bool, error
 */
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func GetCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}

	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "\\", "/", 1)
	}

	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}

	return string(path[0 : i+1]), nil
}

/**
 * 解析文件内容
 * @param filePath string 要解析的文件
 */
func parseFile(filePath, syntax string) error {

	fp, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer fp.Close()

	br := bufio.NewReader(fp)

	for {
		row, _, end := br.ReadLine()

		if end == io.EOF {
			break
		}

		targetSyntax := syntaxMap[syntax]
		if targetSyntax == "yml" {
			parseYamlLine(row)
		} else {
			rowStr := string(row)
			parseLine(rowStr)
		}

	}

	property = nil
	lineNode = LineNode{}

	return nil
}

/**
 * 解析每行数据
 * @param rowStr string 每行数据
 */
func parseLine(rowStr string) {

	// 去除空白，空格等字符
	rowStr = strings.TrimSpace(rowStr)

	// 解析注释行
	if strings.Index(rowStr, "#") == 0 || strings.Index(rowStr, ";") == 0 {
		return
	}

	//匹配节
	if rxFirstSection.MatchString(rowStr) {

		rowStr = rxFirstSection.FindString(rowStr)

		parseSection(rowStr)

		//匹配到节点
	} else if rxNode.MatchString(rowStr) {
		parseProperty(rowStr)
	}
}

/**
 * 解析节
 * @param rowStr string
 */
func parseSection(rowStr string) {

	sectionName = rxSection.ReplaceAllString(rowStr, "")

	property = make(map[string]interface{}) // 重新初始化

	pos := strings.IndexAny(sectionName, ":")

	if pos != -1 {

		child := sectionName[:pos]

		parent := sectionName[pos+1:]

		if child == "" {
			return
		}

		if child == parent {
			return
		}

		// 存在节点直接返回
		_, ok := sections[child]
		if ok {
			return
		}

		//继承父节点
		parentMap = GetSection(parent)

		//设置当前节点
		sections[child] = parentMap

	} else {
		// 存在节点直接返回
		_, ok := sections[sectionName]
		if ok {
			property = GetSection(sectionName)
		}

		sections[sectionName] = property

	}
}

/**
 * 解析节点
 * @param rowStr string
 */
func parseProperty(rowStr string) {

	posEq := strings.IndexAny(rowStr, "=")

	if posEq != -1 {

		keyName := rowStr[:posEq]

		valueStr := rowStr[posEq+1:]

		// 处理等号后面的值
		valueStr = parsNodeValue(valueStr)

		// 处理连续的分隔符
		keyName = parsNodeName(keyName)

		// 处理变量引用
		if strings.HasPrefix(valueStr, "${") {
			if parseVariate(keyName, valueStr) {
				return
			}
		}

		// 处理数组
		if strings.HasPrefix(valueStr, "[") {
			strArr := findSliceString(valueStr)
			if len(strArr) > 0 {
				retSlice := parseSliceRow(valueStr, 0)

				//设置属性
				setProperty(keyName, retSlice)
				//设置值
				property[keyName] = retSlice

				return
			}
		}

		//设置属性
		setProperty(keyName, valueStr)

		//设置值
		property[keyName] = valueStr

	}
}

// 解析变量
func parseVariate(keyName, valueStr string) bool {
	pos := strings.Index(valueStr, "}")
	if pos+1 >= len(valueStr) {
		varStr := rxVariate.FindString(valueStr)
		if varStr != "" && len(varStr) > 3 {
			// 取变量中的值${xxx.xx} => xxx.xx
			quoteKey := varStr[2 : len(varStr)-1]

			var mixVal interface{}
			colonPos := strings.Index(quoteKey, ":")

			if colonPos > 0 {
				fmt.Println(quoteKey[colonPos+1:], quoteKey[:colonPos])
				mixVal = getValBySection(quoteKey[colonPos+1:], quoteKey[:colonPos])
			} else {
				mixVal = getValBySection(quoteKey, "")
			}

			// 设置属性
			setProperty(keyName, mixVal)

			// 设置值
			property[keyName] = mixVal

			return true
		}

		return false
	}

	return false
}

/**
 * 解析节点名
 * @param rowStr string
 * @return string
 */
func parsNodeName(keyName string) string {
	//去掉空格及制表符
	keyName = strings.TrimSpace(keyName)

	//是否含有连续的分隔符，如果有则替换成一个
	if rxSeparator.MatchString(keyName) {
		keyName = rxSeparator.ReplaceAllString(keyName, ".")
	}

	return keyName
}

/**
 * 解析节点值
 * @param rowStr string
 * @return string
 */
func parsNodeValue(valueStr string) string {
	//去掉空格及制表符
	valueStr = strings.TrimSpace(valueStr)

	// 是否是包含单引号 或者双引号  如果是直接取引号中的内容
	if strings.IndexAny(valueStr, "'") != -1 || strings.IndexAny(valueStr, "\"") != -1 {

		//获取引号中的内容，只匹配一次
		valueStr = rxQuotationStart.FindString(valueStr)

		// 是否后面还有引号
		//valueStr = rxQuotationEnd.ReplaceAllString(valueStr, "")
		if len(valueStr) > 0 {
			valueStr = valueStr[1 : len(valueStr)-1]
		}

		// 行内容含有注释信息，则截取掉
	} else if strings.IndexAny(valueStr, "#") != -1 || strings.IndexAny(valueStr, ";") != -1 {

		// 剔除注释后面的内容
		posVal := strings.IndexAny(valueStr, "#")

		if posVal != -1 {
			valueStr = valueStr[:posVal]
			valueStr = strings.TrimSpace(valueStr)
		}

		posVal = strings.IndexAny(valueStr, ";")

		if posVal != -1 {
			valueStr = valueStr[:posVal]
			valueStr = strings.TrimSpace(valueStr)
		}

	}

	return valueStr
}

/**
 * 设置值
 * @param keyName string 节点名
 * @param valueStr string 节点值
 */
func setProperty(keyName string, valueStr interface{}) {

	if strings.IndexAny(keyName, ".") != -1 {

		keyArr := strings.Split(keyName, ".")

		var tempStr = ""

		keyLen := len(keyArr) - 1

		for i := keyLen; i > 0; i-- {

			tempStr = keyArr[i] + "." + tempStr
			tempStr = strings.TrimRight(tempStr, ".")

			if i == keyLen {
				prevStr := strings.Replace(keyName, "."+keyArr[i], "", -1)

				setKeyVal(prevStr, keyArr[i], valueStr)
			} else {
				prevStr := strings.Replace(keyName, "."+tempStr, "", -1)
				currentStr := prevStr + "." + keyArr[i]

				setKeyVal(prevStr, currentStr, property[currentStr])
			}

		}

	} else {
		property[keyName] = valueStr
	}
}

/**
 * 设置链式key值
 * @param prevStr string 父节点名
 * @param currentStr string 当前节点名
 * @param valueStr interface{} 混合类型的值
 */
func setKeyVal(prevStr string, currentStr string, valueStr interface{}) {

	tempObj := property[prevStr]

	if tempObj == nil {

		valMap := make(map[string]interface{})

		valMap[currentStr] = valueStr

		property[prevStr] = valMap

	} else {

		jsonStr, _ := json.Marshal(tempObj)

		tempMap := make(map[string]interface{})

		json.Unmarshal([]byte(jsonStr), &tempMap)

		tempMap[currentStr] = valueStr

		property[prevStr] = tempMap

	}
}

/**
 * 获取节信息
 * @param rowStr string
 * @return map[string]interface{}
 */
func GetSection(sectionKey string) map[string]interface{} {
	// 使用默认值
	if sectionKey == "" {
		sectionKey = defaultName
	}

	property = make(map[string]interface{})

	jsonStr, err := json.Marshal(sections[sectionKey])

	if err != nil {
		sections[sectionKey] = property
	}

	json.Unmarshal([]byte(jsonStr), &property)

	return property
}

// 获取节点内容
func getValBySection(key string, section string) interface{} {
	if section == "" || section == "<nil>" {
		section = defaultName
	}

	if tempRet, ok := sections[section].(map[string]interface{}); ok {
		return tempRet[key]
	}

	return nil
}
