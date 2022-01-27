package global

import (
	"matrixchaindata/pkg/logger"
	"matrixchaindata/pkg/sysinit"
)

var (
	// 全局配置对象
	Config *sysinit.Config
	// 全局日志对象
	GLogger *logger.Logger
	// 全局数据库对象
	GMongodbClient *sysinit.MongoClient
)
