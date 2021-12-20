package scan_server

import (
	"fmt"
	"log"
	"testing"
	"time"
	"xuperdata/global"
	chain_server "xuperdata/internal/chain-server"
	"xuperdata/utils"
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
