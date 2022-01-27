package sysinit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// ----------------- 配置初始化 -----------------

// 配置数据
type Config struct {
	// 运行模式 debug or release
	RunMode string `json:"run_mode"`
	// api服务端口
	HttpPort string `json:"http_port"`
	// 数据库配置
	DB *DBSet `json:"db"`
	// 日志配置
	Log *LogSet
}

// 数据库配置
type DBSet struct {
	// 单节点 or  复制集  single or replicaSet
	ConnMode string `json:"conn_mode"`
	// 用户名
	UserName string `json:"user_name"`
	// 密码
	PassWold string `json:"pass_wold"`
	// 数据库名
	Database string `json:"database"`
	// 服务器
	HostList []string `json:"host_list"`
}

// 日志设置
type LogSet struct {
	// 日志存储目录
	LogPath string `json:"log_path"`
	// 日志文件名字
	LogName string `json:"log_name"`
	// 日志文件格式
	LogExt string `json:"log_ext"`
}

// 配置初始化
// args:
//     - path 配置文件路径
func InitConfig(path string) (*Config, error) {
	config := &Config{}

	// 打开文件并读取内容
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file error, check path")
	}
	defer f.Close()

	configByte, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read file content error")
	}

	if err := json.Unmarshal(configByte, config); err != nil {
		return nil, fmt.Errorf("Unmarshal error")
	}

	return config, nil
}
