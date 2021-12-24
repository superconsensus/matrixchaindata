package service

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	scan_server "matrixchaindata/internal/scan-server"
)

// 存放链相关信息的表 ： chain_info
// { "network":"","node": "", "bcname": ""}
// 网络不同，链名相同 (可以标识)
// 相同网络，节点不同 （区分不了）
// 添加链
// 0 插入数据失败， 1 链已存在， 2 添加链成功
func (s *Serve) AddChain(network, node, bcname string) int {
	opts := options.Update().SetUpsert(true)
	result, err := s.Dao.MongoClient.Collection("chain_info").UpdateOne(
		nil,
		bson.D{
			{"network", network},
			{"node", node},
			{"bcname", bcname},
		},
		bson.D{
			{"$set", bson.D{
				{"network", network},
				{"node", node},
				{"bcname", bcname},
			}},
		},
		opts)

	if err != nil {
		// 插入数据失败
		return 0
	}

	if result.MatchedCount == 1 && result.ModifiedCount == 0 && result.UpsertedCount == 0 {
		// 已经存在
		return 1
	}
	return 2
}

// 根据链名和地址找到信息
func (s *Serve) GetChainInfo(network, bcname string) (bson.M, error) {
	var result bson.M
	//opts := options.FindOne().SetSort(bson.D{{"bcname", -1}})
	err := s.Dao.MongoClient.Collection("chain_info").FindOne(nil,
		bson.D{
			{"network", network},
			{"bcname", bcname},
		}).Decode(&result)

	if err != nil {
		fmt.Println("boxi_2", err)
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}
	return result, nil
}

// 启动扫描服务
func (s *Serve) StartScanService(network, bcname string) error {
	log.Println("start scan")
	// 查找数据库
	chainInfo, err := s.GetChainInfo(network, bcname)
	if err != nil {
		fmt.Println("boxi_1", err)
		return err
	}
	_node := chainInfo["node"].(string)
	_bcname := chainInfo["bcname"].(string)
	_network := chainInfo["network"].(string)
	// 验证一下是否添加了这条记录
	if bcname != _bcname || network != _network {
		return fmt.Errorf("bcname or node error, check it")
	}

	key := fmt.Sprintf("%s_%s", _node, bcname)
	// 检查是否已经启动
	if scan_server.IsExist(key) {
		return fmt.Errorf("%s is scanning", bcname)
	}

	// 启动扫描
	scaner, err := scan_server.NewScanner(_node, bcname)
	if err != nil {
		return fmt.Errorf("new scnanner fail: %v", err)
	}
	// 将扫描记录下来
	// 方便停止扫描
	scan_server.AddScaner(key, scaner)
	// 启动
	err = scaner.Start()
	if err != nil {
		// 启动失败
		// 资源清理
		//1 管理器中移除扫描器
		scan_server.RemoteScanner(key)
		//2 监听器资源清理
		scaner.Watcher.Exit <- struct{}{}
		return fmt.Errorf("start error: %v", err)
	}

	return nil
}

// 停止扫描服务
func (s *Serve) StopScanService(network, bcname string) error {
	log.Println("start scan")
	// 查找数据库
	chainInfo, err := s.GetChainInfo(network, bcname)
	if err != nil {
		fmt.Println("boxi_1", err)
		return err
	}
	_node := chainInfo["node"].(string)

	key := fmt.Sprintf("%s_%s", _node, bcname)
	scanner := scan_server.GetScanner(key)
	if scanner == nil {
		return fmt.Errorf("key error")
	}
	// todo
	// 正在同数据,如果停止
	scanner.Stop()
	// 移除扫描器防止复用
	scan_server.RemoteScanner(key)
	return nil
}

// 查找已经添加的链
func (s *Serve) GetAllChainsInfo() ([]bson.M, error) {
	cursur, err := s.Dao.MongoClient.Collection("chain_info").Find(nil, bson.D{})
	if err != nil {
		return nil, err
	}
	var result []bson.M
	err = cursur.All(nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// 查证正在扫描的链
func (s *Serve) GetScanningChains() []string {
	info := make([]string, 0)
	for key := range scan_server.Scaners {
		info = append(info, key)
	}
	return info
}
