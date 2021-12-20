package chain_server

import (
	"fmt"
	"testing"
	"time"
)

var (
	node   = "120.79.69.94:37102"
	bcname = "xuper"
)

// 测试是否有数据
func Test_Watch(t *testing.T) {

	// 创建grpc连接
	conn := NewConnet(node)
	if conn == nil {
		fmt.Println("conn is nil")
		return
	}
	defer conn.Close()

	watcher, err := WatchBlockEvent(bcname, conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		select {
		case block := <-watcher.FilteredBlockChan:
			fmt.Printf("%#v\n", block)
		case <-time.After(20 * time.Second):
			watcher.Exit <- struct{}{}
			return
		}
	}
}
