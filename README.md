

### 安装
```golang
  go get github.com/stupboy/corn
```

### 参考centos定时写法
- [* * * * * * 秒 分 时 日 周 月]
- [* * * * * * 为每秒执行一次]
- [01 00 12 * * * 为秒为12时00分01秒时执行]
- [1-5 * * * * * 为秒为1-5时，每次都执行]
- [1,3,5 * * * * * 秒为 1 3 5时执行]
- [*/5 * * * * * 为每5秒执行一次]


### 用法
#### AddCorn(方法,定时字符串,方法标志)
#### RunCorn(可选参数，最大携程数量)
```golang
package main

import (
	"github.com/stupboy/corn"
	"log"
)

func init(){
	corn.AddCorn(test,"*/5 * * * * *","test")
	corn.AddCorn(test1,"*/2 * * * * *","test1")
    //corn.RunCorn(500)
	corn.RunCorn()
}

func test(){
	log.Println("aa")
}

func test1(){
	log.Println("bb")
}

func main(){
	select {}
}
```