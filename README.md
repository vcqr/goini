# Go parse ini file [![travis-ci](https://api.travis-ci.org/vcqr/goini.svg)](https://travis-ci.org/vcqr/goini)

可以解析带section的INI文件， 如果不带section，则 section 名默认为 “default”；

一个节可以扩展或者通过在节的名称之后带一个冒号(:)来继承目标节的数据，如果被继承的节点key值重复，则覆盖被继承者的数据；具体可以看下面的例子。

## INI文件(app.ini) Exp：
``` ini

env = prod
[app]
# app name
name = "goini"
port = 8080
location-x = 121.480405
;地理位置
location = 121.480405,31.236221

; 服务器状态
status = "enabled"

[database]
db.driver = mysql
db.host = 127.0.0.1,127.0.0.2
db.port = 3306
db.user = root
db.password = 123456

[redis:database]
db.driver = redis
db.port = 6379

```

Go使用代码(app.go)

``` golang
package main

import (
	"fmt"
	"github.com/vcqr/goini"
)

type DbObj struct {
	Driver   string   `json:"driver"`
	Host     []string `json:"host"`
	Port     int64    `json:"port"`
	User     string   `json:"user"`
	Password string   `json:"password"`
}

func main() {
	config := goini.New()

	// 获取无section的字符串
	fmt.Printf("env=%#v\r\n", config.GetString("env"))

	// 获取指定section的字符串
	fmt.Printf("app.name=%v\r\n", config.GetString("name", "app"))

	// 不存在的key，指定了默认值，则返回指定值
	fmt.Printf("app.null_key=%#v\r\n", config.GetString("null_key", "app", "this is default value"))

	// 获取int类型
	fmt.Printf("app.port=%v\r\n", config.GetInt("port", "app"))

	// 获取float类型
	fmt.Printf("app.location-x=%v\r\n", config.GetFloat("location-x", "app"))

	// 获取Bool类型，部分指定的字符串会转化为bool值
	fmt.Printf("app.status=%v\r\n", config.GetBool("status", "app"))

	// 根据分隔符，转换为指定的切片
	var location []float64
	config.GetSlice("location", ",", &location, "app")
	fmt.Printf("app.location=%v\r\n", location)

	// 转换为指定的Map，map的key数据类型必须是string
	var db map[string]string
	config.GetMap("db", &db, "database")
	fmt.Printf("db=%v\r\n", db)

	// 转换为指定的struct，如果字段是切片类型，可以在Tag中指定分隔符 exp：`json:"demo" ini:"seq=,"`
	var dbObj DbObj
	config.GetStruct("db", &dbObj, "database")
	fmt.Printf("db.mysql=%+v\r\n", dbObj)

	var dbObj2 DbObj
	config.GetStruct("db", &dbObj2, "redis")
	fmt.Printf("db.redis=%+v\r\n", dbObj2)
}

```

请先安装goini
```
go get -u github.com/vcqr/goini
```

编译后可以执行命令
```

./app -c app.ini
```
### 代码输出
``` txt
env="prod"
app.name=goini
app.null_key="this is default value"
app.port=8080
app.location-x=121.480405
app.status=true
app.location=[121.480405 31.236221]
db=map[driver:mysql host:127.0.0.1,127.0.0.2 password:123456 port:3306 user:root]
db.mysql={Driver:mysql Host:[127.0.0.1 127.0.0.2] Port:3306 User:root Password:123456}
db.redis={Driver:redis Host:[127.0.0.1 127.0.0.2] Port:6379 User:root Password:123456}
```

如果在使用过程遇到问题，或者发现bug，或者有更好的建议可以发邮件给我！ 欢迎沟通交流！

# License


Goini is licensed under the 3-Clause BSD License. Goini is 100% free and open-source software.
