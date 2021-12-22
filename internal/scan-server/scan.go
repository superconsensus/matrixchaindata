package scan_server

import (
	"fmt"
	"google.golang.org/grpc"
	"log"
	"matrixchaindata/global"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/utils"
)

// 扫描器
type Scaner struct {
	// 链相关
	Node     string
	Bcname   string
	GrpcConn *grpc.ClientConn
	// 数据库
	DBWrite *WriteDB
	// 监听器用于获取数据
	Watcher *chain_server.Watcher
	// exit
	Exit chan struct{}
}

// 创建扫描器
func NewScanner(node, bcname string) (*Scaner, error) {
	// 创建grpc连接
	conn := chain_server.NewConnet(node)
	if conn == nil {
		return nil, fmt.Errorf("conn fail")
	}

	//数据库处理
	// 传入的是全局db,不要在这里关闭。应该在mian中处理
	writeDB := NewWriterDB(global.GloMongodbClient)

	// 监听数据
	watcher, err := chain_server.WatchBlockEvent(bcname, conn)
	if err != nil {
		return nil, err
	}

	return &Scaner{
		Node:     node,
		Bcname:   bcname,
		GrpcConn: conn,
		DBWrite:  writeDB,
		Watcher:  watcher,
		Exit:     make(chan struct{}),
	}, nil
}

// 停止扫描工作
// 关闭工作是这样的，连接不为空，则发送退出信号
// goroutin 接收到信息，则关闭grpc连接，在退出
func (s *Scaner) Stop() {
	if s.GrpcConn != nil {
		s.Exit <- struct{}{}
	}
}

// 启动扫描工作
func (s *Scaner) Start() error {
	//1 获取数据库中缺少的区块
	// 处理此刻之前的数据
	//err := s.DBWrite.Init(s.Node, s.Bcname)
	//if err != nil {
	//	return err
	//}
	// 尝试使用goroutine 找出缺少可块
	// ? 当区块达到10万级别的时候处理起来十分慢
	// 尝试goroutine异步执行
	go func() {
		//defer func() {
		//	log.Println("获取数据库中缺少的区块error")
		//	// 这里出错需要处理scan goroutine？
		//	// 需要处理
		//	//s.Stop()
		//	log.Println("try recover")
		//	if err := recover(); err != nil {
		//		log.Println("catch error:", err)
		//	}
		//}()
		err := s.DBWrite.HandleLackBlocks(s.Node, s.Bcname)
		if err != nil {
			log.Println(err)
			// 这里直接清理资源
		}
	}()

	//2 处理新数据
	go func() {
		defer func() {
			// 监听，如果出现错误关闭管道并退出
			// 通知监听goroutine退出，随后自己停止
			s.Watcher.Exit <- struct{}{}
			s.GrpcConn.Close()
			return
		}()
		for {
			select {
			case <-s.Exit:
				log.Println("stop scnner")
				// 通知监听器退出
				s.Watcher.Exit <- struct{}{}
				// 关闭grpc连接
				s.GrpcConn.Close()
				log.Println("clear network source")
				return
			case block := <-s.Watcher.FilteredBlockChan:
				//log.Println("get data")
				// 处理数据
				err := s.DBWrite.Save(utils.FromInternalBlockPB(block), s.Node, s.Bcname)
				if err != nil {
					log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
				}
			}
		}
	}()

	return nil
}
