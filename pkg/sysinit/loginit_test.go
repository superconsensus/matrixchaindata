package sysinit

import "testing"

func Test_InitLogger(t *testing.T) {
	// 读取配置
	filepath := "./mock/config_single.json"
	config, _ := InitConfig(filepath)

	/// 测试logger
	// 新建
	logger := InitLogger(config.Log)
	// 使用
	logger.Info("test logger")
}
