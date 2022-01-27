package sysinit

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"matrixchaindata/pkg/logger"
)

// 日志初始化
func InitLogger(logSet *LogSet) *logger.Logger {
	// 拼接日志文件名字
	fileName := logSet.LogPath + "/" + logSet.LogName + logSet.LogExt
	l := logger.NewLogger(&lumberjack.Logger{
		Filename:  fileName,
		MaxSize:   600,
		MaxAge:    10,
		LocalTime: true,
	}, "", log.LstdFlags).WithCallers(5)
	return l
}
