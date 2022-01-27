package sysinit

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"strings"
	"time"
)

// 数据库初始化

//Replica Sets复制集
//MongoDB 在 1.6 版本对开发了新功能replica set，这比之前的replication 功能要强大一 些，
//增加了故障自动切换和自动修复成员节点，各个DB 之间数据完全一致，大大降低了维 护成功。
//auto shard 已经明确说明不支持replication paris，建议使用replica set，replica set 故障切换完全自动。
//
//Replica Sets的结构类似一个集群，完全可以把它当成一个集群，因为它确实与集群实现的作用是一样的：如果其中一个节点出现故障，其他节点马上会将业务接管过来而无须停机操作

func InitDB(dbSet *DBSet) (*MongoClient, error) {
	// 连接模式
	if dbSet.ConnMode == "single" {
		// 单个mongo服务器
		if len(dbSet.HostList) <= 0 {
			log.Println("host error, check it")
			return nil, fmt.Errorf("host error, check it")
		}
		mongoUrl := "mongodb://" + dbSet.UserName + ":" + dbSet.PassWold + "@" + dbSet.HostList[0]
		return NewMongoClient(mongoUrl, dbSet.Database)
	} else if dbSet.ConnMode == "replicaSet" {
		// todo 未测试
		// 复制集模式
		//uri: mongodb://dev:dev123@192.168.1.51:27017,192.168.1.52:27017,192.168.1.53:27017
		//database: wd_temp_test
		hosts := strings.Join(dbSet.HostList, ",")
		mongoUrl := "mongodb://" + dbSet.UserName + ":" + dbSet.PassWold + "@" + hosts
		return NewMongoClient(mongoUrl, dbSet.Database)
	} else {
		// 连接模式出错
		log.Println(" conn_mode parm error, check them")
	}
	return nil, nil
}

// mongodb的客户端
type MongoClient struct {
	*mongo.Client
	*mongo.Database
}

// NewMongoClient 创建
func NewMongoClient(dataSource, database string) (*MongoClient, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(dataSource).SetConnectTimeout(10 * time.Second))
	if err != nil {
		return nil, err
	}
	err = client.Connect(context.Background())
	if err != nil {
		return nil, err
	}
	return &MongoClient{client, client.Database(database)}, nil
}

func (m *MongoClient) Close() error {
	return m.Client.Disconnect(nil)
}
