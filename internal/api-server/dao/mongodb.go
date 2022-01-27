package dao

import (
	"matrixchaindata/pkg/sysinit"
)

type Dao struct {
	// 数据库客户端
	MongoClient *sysinit.MongoClient
}

// 新建数据连接实例
func NewDao(mongoclient *sysinit.MongoClient) *Dao {
	return &Dao{
		MongoClient: mongoclient,
	}
}

func (d *Dao) Close() {
	d.MongoClient.Close()
}
