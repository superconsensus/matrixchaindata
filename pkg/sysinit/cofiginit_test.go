package sysinit

import (
	"fmt"
	"testing"
)

// 测试配置读取情况
func Test_InitConfig(t *testing.T) {
	filepath := "./mock/config_single.json"
	config, _ := InitConfig(filepath)
	fmt.Printf("%#v\n", config)
	fmt.Printf("%#v\n", config.DB)
	fmt.Printf("%#v\n", config.Log)

	for _, v := range config.DB.HostList {
		fmt.Println(v)
	}
}
