package main

import (
	"log"
	"matrixchaindata/global"
	"matrixchaindata/internal/api-server/router"
	"matrixchaindata/pkg/settings"
	"os"
)

func main() {
	//读取配置文件
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	settings.ParseConfig(dir + "/config/config.json")
	log.Printf("%#v", settings.Setting)

	// 实例化数据库
	err = global.InitmongoDB(settings.Setting.MongoDB, settings.Setting.Database)
	if err != nil {
		log.Println(err)
	}

	//注册路由
	r := router.InitRouter()
	_ = r.Run(settings.Setting.HttpPort)
	//s := &http.Server{
	//	Addr:           settings.Setting.HttpPort,
	//	Handler:        r,
	//	ReadTimeout:    60,
	//	WriteTimeout:   60,
	//	MaxHeaderBytes: 1 << 20,
	//}
	//
	//// 启动http server
	//go func() {
	//	if err := s.ListenAndServe(); err != nil {
	//		log.Println("listen err :", err)
	//	}
	//}()
	//
	//// 优雅停止
	//quit := make(chan os.Signal)
	//signal.Notify(quit, os.Interrupt)
	//<-quit
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//if err := s.Shutdown(ctx); err != nil {
	//	log.Fatal("Server Shutdown:", err)
	//}
	//log.Println("Server exiting")
}
