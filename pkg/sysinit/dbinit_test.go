package sysinit

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"testing"
)

// 测试 dbinit
func Test_InitDB_Single(t *testing.T) {
	// 读取配置
	filepath := "./mock/config2.json"
	config, _ := InitConfig(filepath)

	//测试single数据库连接
	client, err := InitDB(config.DB)
	if err != nil {
		log.Println("init db error")
		return
	}
	defer client.Close()

	result, err := client.ListDatabases(nil, bson.D{})
	if err != nil {
		log.Println("list databases error", err)
		return
	}
	for _, v := range result.Databases {
		fmt.Println(v)
	}
}

// todo 完成复制集测试
// 没有搭建复制集合，测试是失败的
func Test_InitDB_ReplicaSet(t *testing.T) {
	// 读取配置
	filepath := "./mock/config_single.json"
	config, _ := InitConfig(filepath)

	//测试ReplicaSet数据库连接
	client, err := InitDB(config.DB)
	if err != nil {
		log.Println("init db error")
		return
	}
	defer client.Close()

	result, err := client.ListDatabases(nil, bson.D{})
	if err != nil {
		log.Println("list databases error", err)
		return
	}
	for _, v := range result.Databases {
		fmt.Println(v)
	}
}
