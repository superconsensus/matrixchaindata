package scan_server

import (
	"fmt"
	"google.golang.org/grpc"
	"log"
	"matrixchaindata/global"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/utils"
)

// 扫描指定链---将数据存进数据库
// 开启goroutine 每一条链分配一个
//func Scan(DataSource, Database, NodeIp, Bcname, filter string) error {
//	//初始化数据库连接
//	var err error
//	mongoClient, err := global.NewMongoClient(DataSource, Database)
//	if err != nil {
//		log.Println("can't connecting to mongodb, err:", err)
//		return nil
//	}
//	writeDB := NewWriterDB(mongoClient)
//
//	defer  writeDB.Cloce()
//
//	//fmt.Println(c.Restore)
//	////清空数据库
//	//if c.Restore {
//	//	err = mongoClient.Drop(nil)
//	//	if err != nil {
//	//		err = mongoClient.Database.Collection("count").Drop(nil)
//	//		err = mongoClient.Database.Collection("account").Drop(nil)
//	//		err = mongoClient.Database.Collection("tx").Drop(nil)
//	//		err = mongoClient.Database.Collection("block").Drop(nil)
//	//		if err != nil {
//	//			log.Printf("clean the database failed, error: %v", err)
//	//			return err
//	//		}
//	//	}
//	//	fmt.Println("clean the database successed")
//	//}
//
//	//data, err := ioutil.ReadFile(c.DescFile)
//	//if err != nil {
//	//	fmt.Println("subscribe failed, error:", err)
//	//	return nil
//	//}
//	conn, err := grpc.Dial(NodeIp, grpc.WithInsecure())
//	if err != nil {
//		fmt.Println("unsubscribe failed, err msg:", err)
//		return nil
//	}
//	defer conn.Close()
//
//	filter1 := &pb.BlockFilter{
//		Bcname: Bcname,
//	}
//	//bcname = c.nodeName
//	//node = c.DestIP
//	err2 := json.Unmarshal([]byte(filter), filter1)
//	if err2 != nil {
//		return err2
//	}
//
//	buf, _ := proto.Marshal(filter1)
//	request := &pb.SubscribeRequest{
//		Type:   pb.SubscribeType_BLOCK,
//		Filter: buf,
//	}
//
//	err = writeDB.Init(NodeIp, Bcname) //获取数据库中缺少的区块
//	if err != nil {
//		log.Fatalf("get lack blocks failed, error: %s", err)
//	}
//
//	xclient := pb.NewEventServiceClient(conn)
//	stream, err := xclient.Subscribe(context.TODO(), request)
//	if err != nil {
//		return err
//	}
//	for {
//		event, err := stream.Recv()
//		if err == io.EOF {
//			return nil
//		}
//		if err != nil {
//			return err
//		}
//		var block pb.InternalBlock
//		err = proto.Unmarshal(event.Payload, &block)
//		if err != nil {
//			return err
//		}
//		//if len(block.GetTxs()) == 0 && c.skipEmptyTx {
//		//	continue
//		//}
//		//if c.ShowBlock {
//		//	fmt.Println("Recv block:", block.Height)
//		//}
//		//存数据
//		err = writeDB.Save(utils.FromInternalBlockPB(&block), NodeIp, Bcname)
//		if err != nil {
//			log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
//		}
//
//		//c.printBlock(&block)
//	}
//	return nil
//}

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
	//err := s.DBWrite.Init(s.Node, s.Bcname)
	//if err != nil {
	//	return err
	//}

	// 尝试使用goroutine 找出缺少可块
	// ? 当区块达到10万级别的时候处理起来十分慢
	// 尝试goroutine异步执行
	go func() {
		defer func() {
			fmt.Println("获取数据库中缺少的区块 error")
			// 这里出错需要处理scan goroutine
			// (?)
			s.Stop()
			return
		}()
		err := s.DBWrite.Init(s.Node, s.Bcname)
		if err != nil {
			log.Println(err)
		}
	}()

	//2 处理数据
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
				// 通知监听器退出
				s.Watcher.Exit <- struct{}{}
				// 关闭grpc连接
				s.GrpcConn.Close()
				return
			case block := <-s.Watcher.FilteredBlockChan:
				log.Println("get data")
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

// 扫描数据处理
// 目前需要处理的问题是 grpc优雅关闭问题
// 对scan 使用goroutine, 做个监听
//func Scan(node, bcname string, watcher *chain_server.Watcher,writeDB *WriteDB) error {
//	go func() {
//		defer func() {
//			// 监听，如果出现错误关闭管道并退出
//			// 通知监听goroutine 退出，随后自己停止
//			watcher.Exit <- struct{}{}
//			return
//		}()
//		for {
//			select {
//			case block := <-watcher.FilteredBlockChan:
//				log.Println("get data")
//				// 处理数据
//				err = writeDB.Save(utils.FromInternalBlockPB(block), node, bcname)
//				if err != nil {
//					log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
//				}
//			}
//		}
//	}()
//	return nil
//}
