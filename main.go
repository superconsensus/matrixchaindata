package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"xuperdata/global"
	"xuperdata/internal/api-server/router"
	"xuperdata/settings"
)


func main()  {
	//读取配置文件
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	settings.ParseConfig(dir+"/config/config.json")
	fmt.Printf("%#v", settings.Setting)

	// 实例化数据库
	err = global.InitmongoDB(settings.Setting.MongoDB, settings.Setting.Database)
	if err != nil {
		fmt.Println(err)
		fmt.Println("init db fail")
	}
	//实例化gin框架
	r := gin.Default()

	//注册路由
	router.InitRouter(r)
	_ = r.Run(settings.Setting.HttpPort) // listen and serve on 0.0.0.0:8080
}
