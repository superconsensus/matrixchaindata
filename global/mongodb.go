package global

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

var GloMongodbClient  *MongoClient

// 创建全局的配置
func InitmongoDB(dataSource, database string) error {
	client, err := NewMongoClient(dataSource, database)
	if err != nil {
		return fmt.Errorf("init db error")
	}
	GloMongodbClient = client
	return nil
}

// mongodb的客户端
type MongoClient struct {
	*mongo.Client
	*mongo.Database
}

// NewMongoClient 创建
func NewMongoClient(dataSource, database string) (*MongoClient, error) {
	client, err := mongo.NewClient(options.Client().
		ApplyURI(dataSource).
		SetConnectTimeout(10 * time.Second))
	if err != nil {
		return nil, err
	}

	//ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	//defer client.Disconnect(ctx)

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	//databases, err := client.ListDatabaseNames(ctx, bson.M{})
	//if err != nil {
	//	return nil, err
	//}
	//fmt.Println(databases)

	return &MongoClient{client, client.Database(database)}, nil
}

func (m *MongoClient) Close() error {
	return m.Client.Disconnect(nil)
}


