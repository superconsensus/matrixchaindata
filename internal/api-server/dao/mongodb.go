package dao

import (
	"matrixchaindata/global"
)

type Dao struct {
	// 数据库客户端
	MongoClient *global.MongoClient
}

// 新建数据连接实例
func NewDao(mongoclient *global.MongoClient) *Dao {
	return &Dao{
		MongoClient: mongoclient,
	}
}

func (d *Dao) Close() {
	d.MongoClient.Close()
}
