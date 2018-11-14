# Go parse ini file
可以解析带section的INI文件， 如果不带section，则 section 名默认为 “default”；

一个节可以扩展或者通过在节的名称之后带一个冒号(:)来继承目标节的数据，如果被继承的节点key值重复，则覆盖被继承者的数据；具体可以看下面的例子。

## INI文件(app.ini) Exp：
``` ini
[app]
#生产环境
env = prod

# server 配置
server.host = 127.0.0.1
server.port = 8080

# db 配置
; 驱动
db.driver = "com.mysql.jdbc.Driver"
; 数据连接
db.url = "jdbc:mysql://localhost:3306/test";
; 用户名
db.user = "root"
; 密码
db.password = "123456"

# sever1继承app配置
[server1:app]
env = dev

server.port = 8081
; 数据连接
db.url = "jdbc:mysql://localhost:3307/server1";
; 用户名
db.user = "server"
; 密码
db.password = "qwerty"

# sever2继承app配置
[server2:server1]
env = qa
server.port = 8082

```

Go使用代码(app.go)

``` golang
package main

import (
	"fmt"
	"goini/goini"
)

func main() {
	config := goini.New()

	// 获取app的当前环境
	fmt.Println("app.env = ", config.GetValBySection("env", "app"))

	// 获取server1的当前环境
	fmt.Println("server1.env = ", config.GetValBySection("env", "server1"))

	// 获取server2的当前环境
	fmt.Println("server1.env = ", config.GetValBySection("env", "server2"))

	// 获取server1的当前server配置
	fmt.Println("server1.server.port = ", config.GetValBySection("server.port", "server1"))

	// 获取server2的当前server配置
	fmt.Println("server2.server.port = ", config.GetValBySection("server.port", "server2"))

	fmt.Println("server2.server = ", config.GetValBySection("server", "server2"))
	fmt.Println("server2.db = ", config.GetValBySection("db", "server2"))

	//添加新节
	config.SetSection("server3:app")
	//添加节点值
	config.SetValBySection("env", "local", "server3")

	fmt.Println("server3.env = ", config.GetValBySection("env", "server3"))
	fmt.Println("server3.db.url = ", config.GetValBySection("db.url", "server3"))
}

编译后可以执行命令

```
./go -c app.ini
```

```
### 代码输出
``` txt
app.env =  prod
server1.env =  dev
server1.env =  qa
server1.server.port =  8081
server2.server.port =  8082
server2.server =  map[host:127.0.0.1 port:8082]
server2.db =  map[driver:com.mysql.jdbc.Driver password:qwerty url:jdbc:mysql://localhost:3307/server1 user:server]
server3.env =  local
server3.db.url =  jdbc:mysql://localhost:3306/test
```

如果在使用过程遇到问题，或者发现bug，或者有更好的建议可以发邮件给我！ 欢迎沟通交流！
