package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var (
	// 全局日志
	Setting *Config
)

type Config struct {
	Node string `json:"node"`
	HttpPort string `json:"http_port"`
	MongoDB string `json:"mongodb"`
	Database string `json:"database"`
}

func ParseConfig(path string) *Config {
	conf := new(Config)

	//读取配置文件
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(path)
		panic("打开配置文件出错")
	}
	defer file.Close()

	confByte, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		panic("读取配置文件出错")
	}

	if err := json.Unmarshal(confByte, conf); err != nil {
		fmt.Println(err)
		panic("解析配置文件出错")
	}
	Setting = conf
	return conf
}