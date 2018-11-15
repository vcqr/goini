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
	"regexp"
	"runtime"
	"strings"
)

type Config interface {
	//获取值，默认取default节下的节点取值
	Get(key string) interface{}

	//设置值，默认从default节下节点设置
	Set(key string, val interface{})

	// 使用节，获取相关节点的值
	GetValBySection(key string, section string) interface{}

	// 使用节，设置相关节点的值
	SetValBySection(key string, val interface{}, section string)

	// 设置节
	SetSection(section string)

	// 取节值
	GetSection(section string) map[string]interface{}
}

type Goini struct {
	filePath string
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
var defaultName = "default"

// 记录当前节名
var sectionName string

/**
 * 默认节点取值
 * @param key string 节点名称
 * @return interface{}
 */
func (goini *Goini) Get(key string) interface{} {
	return goini.GetValBySection(key, "")
}

/**
 *  设置默认节点值
 * @param key string 节点名称
 * @param val string 值
 */
func (goini *Goini) Set(key string, val string) {
	goini.SetValBySection(key, val, "")
}

/**
 *  获取指定节内容
 * @param section string 节名
 * @return map[string]interface{}
 */
func (goini *Goini) GetSection(section string) map[string]interface{} {
	return GetSection(section)
}

/**
 *  新增节
 * @param section string 节名
 */
func (goini *Goini) SetSection(section string) {
	//设置节点
	parseSection(section)
}

/**
 *  根据节和节点名称获取值
 * @param key string 节点名
 * @param section string 节名
 * @return interface{} 混合类型内容
 */
func (goini *Goini) GetValBySection(key string, section string) interface{} {
	if section == "" {
		section = defaultName
	}

	tempRet := make(map[string]interface{})
	jsonStr, err := json.Marshal(sections[section])

	if err != nil {
		return nil
	}

	json.Unmarshal([]byte(jsonStr), &tempRet)

	return tempRet[key]
}

/**
 *  根据节和节点名设置值
 * @param key string 节点名
 * @param val string 值
 * @param section string 节名
 */
func (goini *Goini) SetValBySection(key string, val string, section string) {
	if section == "" {
		section = defaultName
	}

	if key == "" {
		panic("set node error: key cannot be nil")
	}

	//设置节点
	parseSection(section)

	// 设置属性
	parseProperty(key + "=" + val)

}

/**
 *  构造对象
 * @return *Goini
 */
func New() *Goini {
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

	// 文件是否存在
	isFile, _ := PathExists(path)
	if !isFile {
		panic("goini error: application.ini file cannot be exists")
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
	parseFile(config.filePath)

	return config

}

/**
 *  加载指定文件
 * @param path string 文件路径
 * @return *Goini
 */
func SetCofig(path string) *Goini {
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
	parseFile(config.filePath)

	return config
}

/**
 *  获取命令行中的文件路径
 * @return string
 */
func ArgConfigPath() string {
	flag.Parse()

	path := *iniPath
	if path != "" {
		return path
	}

	path = *confPath

	return path
}

/**
 *  文件是否真实存在
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
 *  解析文件内容
 * @param filePath string 要解析的文件
 */
func parseFile(filePath string) {

	fp, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	defer fp.Close()

	br := bufio.NewReader(fp)

	for {
		row, _, end := br.ReadLine()

		if end == io.EOF {
			break
		}

		rowStr := string(row)

		parseLine(rowStr)

	}

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

	// 节匹配
	// 只解析第一个符合规范的节名
	regSection := regexp.MustCompile(`(?U)(\[.*\])`)

	// 节点匹配
	regNode := regexp.MustCompile(`.*=.*`)

	//匹配节
	if regSection.MatchString(rowStr) {

		rowStr = regSection.FindString(rowStr)

		parseSection(rowStr)

		//匹配到节点
	} else if regNode.MatchString(rowStr) {
		parseProperty(rowStr)
	}

}

/**
 * 解析节
 * @param rowStr string
 */
func parseSection(rowStr string) {

	regTemp := regexp.MustCompile(`\s+|\[|\]|\*+`)

	sectionName = regTemp.ReplaceAllString(rowStr, "")

	property = make(map[string]interface{}) // 重新初始化

	pos := strings.IndexAny(sectionName, ":")

	if pos != -1 {

		sectionName = string([]rune(sectionName))

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

		//jsonStr, _ := json.Marshal(parentMap)
		//tempMap := make(map[string]interface{})
		//json.Unmarshal([]byte(jsonStr), &tempMap)
		//property = parentMap

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

		rowStr = string([]rune(rowStr))

		keyName := rowStr[:posEq]

		valueStr := rowStr[posEq+1:]

		// 处理等号后面的值
		valueStr = parsNodeValue(valueStr)

		// 处理连续的分隔符
		keyName = string([]rune(keyName))
		keyName = parsNodeName(keyName)

		//设置属性
		setProperty(keyName, valueStr)

		//设置值
		property[keyName] = valueStr

	}

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
	tempReg := regexp.MustCompile(`(\.\.+)`)

	if tempReg.MatchString(keyName) {
		keyName = tempReg.ReplaceAllString(keyName, ".")
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
		tempReg := regexp.MustCompile(`(?U)(".*"|'.*')`)

		valueStr = tempReg.FindString(valueStr)

		//
		tempReg = regexp.MustCompile(`"|'`)
		valueStr = tempReg.ReplaceAllString(valueStr, "")

		// 行内容含有注释信息，则截取掉
	} else if strings.IndexAny(valueStr, "#") != -1 || strings.IndexAny(valueStr, ";") != -1 {

		// 剔除注释后面的内容
		posVal := strings.IndexAny(valueStr, "#")
		valueStr = string([]rune(valueStr))

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
func setProperty(keyName string, valueStr string) {

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
