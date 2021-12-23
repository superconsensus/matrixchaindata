package scan_server

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"matrixchaindata/global"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/utils"
	"testing"
	"time"
)

// 测试是否可以将数据写入数据库
func Test_Scan(t *testing.T) {
	dbSource := "mongodb://admin:admin@192.168.199.124:27017"
	database := "boxi"
	node := "120.79.69.94:37102"
	bcname := "xuper"

	dbClient, err := global.NewMongoClient(dbSource, database)
	if err != nil {
		fmt.Println("create db clien error")
	}
	writeDB := NewWriterDB(dbClient)

	// 监听数据
	conn := chain_server.NewConnet(node)
	if conn == nil {
		fmt.Println("conn is nil")
		return
	}
	watcher, err := chain_server.WatchBlockEvent(bcname, conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 开启goroutine 处理数据
	go func() {
		defer func() {
			// 监听，如果出现错误关闭管道并退出
			// 通知监听goroutine 退出，随后自己停止
			watcher.Exit <- struct{}{}
			return
		}()
		for {
			select {
			case block := <-watcher.FilteredBlockChan:
				log.Println("get data")
				// 处理数据
				err = writeDB.Save(utils.FromInternalBlockPB(block), node, bcname)
				if err != nil {
					log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
				}
			}
		}
	}()
	time.Sleep(120 * time.Second)
}

func TestWriteDB_IsHandle(t *testing.T) {
	dbSource := "mongodb://admin:admin@192.168.199.128:27017"
	database := "boxi"
	//node := "120.79.69.94:37102"
	bcname := "xuper"
	//block_id := 1440092

	dbClient, err := global.NewMongoClient(dbSource, database)
	if err != nil {
		fmt.Println("create db clien error")
	}
	//blockCol := dbClient.Database.Collection(fmt.Sprintf("block_%s", bcname))
	//
	//data := blockCol.FindOne(nil, bson.D{{"_id", block_id}})
	//if data.Err() != nil {
	//	fmt.Println(data.Err())
	//} else {
	//	fmt.Println("has data")
	//}
	start := time.Now()
	blockCol := dbClient.Collection(fmt.Sprintf("block_%s", bcname))

	//获取数据库中最后的区块高度
	sort := 1
	limit := int64(0)
	var heights []int64

	cursor, err := blockCol.Find(nil, bson.M{}, &options.FindOptions{
		Projection: bson.M{"_id": 1},
		Sort:       bson.M{"_id": sort},
		Limit:      &limit,
	})

	if err != nil && err != mongo.ErrNoDocuments {
		fmt.Println()
		return
	}
	var reply bson.A
	if cursor != nil {
		err = cursor.All(nil, &reply)
	}
	fmt.Println(len(reply))
	//获取需要遍历的区块高度
	heights = make([]int64, len(reply))
	for i, v := range reply {
		heights[i] = v.(bson.D).Map()["_id"].(int64)
	}
	//lacks := findLacks(heights)
	fmt.Println(time.Now().Sub(start).String())
	fmt.Println(heights[len(heights)-1])
	//fmt.Println(len(heights), len(lacks))
}
