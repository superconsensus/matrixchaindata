package service

import (
	"matrixchaindata/global"
	"matrixchaindata/internal/api-server/dao"
)

type Serve struct {
	Dao *dao.Dao
}

func NewSever() *Serve {
	// 直接从全局的db拿到 数据库连接构造dao层
	d := dao.NewDao(global.GloMongodbClient)
	return newServe(d)
}

func newServe(d *dao.Dao) *Serve {
	return &Serve{
		Dao: d,
	}
}
