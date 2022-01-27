package main

import (
	"log"
	"matrixchaindata/global"
	"matrixchaindata/internal/api-server/router"
	"matrixchaindata/pkg/sysinit"
	"os"
)

// 程序初始化
func SysInit() error {
	log.Println("------------start config init ------------")
	// 配置初始化
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	config, err := sysinit.InitConfig(dir + "/config/config_single.json")
	if err != nil {
		return err
	}
	global.Config = config

	log.Println("------------ start db init ------------")
	// 数据库配置初始化
	client, err := sysinit.InitDB(global.Config.DB)
	if err != nil {
		log.Println("init db error, check them")
		return err
	}
	global.GMongodbClient = client

	// 日志配置初始化
	//global.GLogger = sysinit.InitLogger(global.Config.Log)
	return nil
}

func main() {
	// 程序初始化
	err := SysInit()
	if err != nil {
		log.Println("sysinit error, check them", err)
		return
	}
	//注册路由
	r := router.InitRouter()
	_ = r.Run(global.Config.HttpPort)
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
