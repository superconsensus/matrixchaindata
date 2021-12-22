package scan_server

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/wxnacy/wgo/arrays"
	"github.com/xuperchain/xuperchain/service/pb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"log"
	"matrixchaindata/global"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/utils"
	"sync"
	"time"
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
	// 接受数据管道（接受监听数据 + 接受缺少区块的数据 ）
	LackBlockChan chan *pb.InternalBlock
	// exit 退出管道
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
		Node:          node,
		Bcname:        bcname,
		GrpcConn:      conn,
		DBWrite:       writeDB,
		Watcher:       watcher,
		LackBlockChan: make(chan *pb.InternalBlock, 100),
		Exit:          make(chan struct{}),
	}, nil
}

// 停止扫描工作
// 关闭工作是这样的，连接不为空，则发送退出信号
// goroutin 接收到信息，则关闭grpc连接，在退出
// todo 优化
func (s *Scaner) Stop() {
	s.Exit <- struct{}{}
}

// 启动扫描工作
// 先扫描数据库中缺少的数据
// 然后再获取监听到的数据
func (s *Scaner) Start() error {
	//1 获取数据库中缺少的区块
	// 把获取的区块写入LackBlockChan 管道中
	// 这样可以避免并发写数据库
	go func() {
		//defer func() {
		//	log.Println("try recover")
		//	if err := recover(); err != nil {
		//		log.Println("catch error:", err)
		//	}
		//}()
		err := s.HandleLackBlocks()
		if err != nil {
			log.Println(err)
			// 这里直接清理资源
		}
	}()

	//2 处理数据
	// 监听两个管道，缺少区块管道和订阅区块的管道
	// 最后写入数据库
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
				// 推出管道是在监听退出信号的
				// 在退出的时候开始清理一些资源
				log.Println("stop scnner")
				// 通知监听器退出
				s.Watcher.Exit <- struct{}{}
				// 关闭grpc连接
				s.GrpcConn.Close()
				log.Println("clear network source")
				return
			case block := <-s.Watcher.FilteredBlockChan:
				// 处理监听器中数据
				err := s.DBWrite.Save(utils.FromInternalBlockPB(block), s.Node, s.Bcname)
				if err != nil {
					log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
				}
			case lackBlock, ok := <-s.LackBlockChan:
				// 处理缺少区块管道中的数据
				// 在完成的时候关闭管道
				// 防止不断读取关闭的管道
				if ok {
					err := s.DBWrite.Save(utils.FromInternalBlockPB(lackBlock), s.Node, s.Bcname)
					if err != nil {
						log.Printf("save block to mongodb failed, height: %d, error: %s", lackBlock.Height, err)
					}
				} else {
					s.LackBlockChan = nil
				}
			default:
				//没有监听到数据睡眠 1秒
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()

	return nil
}

// 扫描缺少的区块
func (s *Scaner) HandleLackBlocks() error {
	// 最新高度
	_, height, err := chain_server.GetUtxoTotalAndTrunkHeight(s.Node, s.Bcname)
	if err != nil {
		return err
	}

	// 获取缺少的区块
	return s.GetLackBlocks(&utils.InternalBlock{
		Height: height,
	})
}

// 获取最新区块之前的block
// 读了两次数据库
// 直接
func (s *Scaner) GetLackBlocks(block *utils.InternalBlock) error {
	log.Println("start get lack blocks")
	// 获取区块集合
	blockCol := s.DBWrite.MongoClient.Database.Collection(fmt.Sprintf("block_%s_%s", s.Node, s.Bcname))

	//获取数据库中最后的区块高度
	sort := -1
	limit := int64(1)
	var heights []int64

again:
	{
		cursor, err := blockCol.Find(nil, bson.M{}, &options.FindOptions{
			Projection: bson.M{"_id": 1},
			Sort:       bson.M{"_id": sort},
			Limit:      &limit,
		})

		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}
		var reply bson.A
		if cursor != nil {
			err = cursor.All(nil, &reply)
		}
		//fmt.Println("reply:", reply)

		//获取需要遍历的区块高度
		heights = make([]int64, len(reply))
		for i, v := range reply {
			heights[i] = v.(bson.D).Map()["_id"].(int64)
		}
		//fmt.Println("heights:", heights)
	}

	//高度不匹配,找出缺少的区块高度，并获取区块
	if len(heights) == 1 && heights[0] != block.Height-1 {
		sort = 1         //顺序排列
		limit = int64(0) //获取所有区块
		goto again
	}

	//添加一个值,避免空指针异常
	heights = append(heights, block.Height)
	//找到缺少的区块
	lacks := findLacks(heights)

	//用个协程池,避免控制并发量
	defer ants.Release()
	wg := sync.WaitGroup{}
	p, _ := ants.NewPoolWithFunc(gosize, func(i interface{}) {
		func(height int64) {
			iblock, err := chain_server.GetBlockByHeight(s.Node, s.Bcname, height)
			if err != nil {
				log.Printf("get block by height failed, height: %d, error: %s", height, err)
				return
			}
			s.LackBlockChan <- iblock
		}(i.(int64))
		wg.Done()
	})
	defer p.Release()
	log.Println("缺少区块的数量：", len(lacks))
	for _, height := range lacks {
		//fmt.Println("start get lack block:", height)
		wg.Add(1)
		_ = p.Invoke(height)
	}

	wg.Wait()
	log.Println("get lack blocks finished")
	return nil
}

// 找出缺少的区块
func findLacks(heights []int64) []int64 {
	log.Printf("mongodb's blocks size: %d", len(heights))

	if len(heights) == 0 {
		return nil
	}

	lacks := make([]int64, 0)

	var i int64 = 0
	for ; i < heights[len(heights)-1]; i++ {
		//不存在,记录该值
		index := arrays.ContainsInt(heights, i)
		if index == -1 {
			lacks = append(lacks, i)
			continue
		}
		//存在,剔除该值
		heights = append(heights[:index], heights[index+1:]...)
		//fmt.Println("heights:", heights)
	}
	log.Printf("lack blocks size: %d", len(lacks))
	return lacks
}

// todo test
func (s *Scaner) getLackBlocks(block *utils.InternalBlock) error {
	log.Println("start get lack blocks")
	// 获取区块集合
	blockCol := s.DBWrite.MongoClient.Database.Collection(fmt.Sprintf("block_%s_%s", s.Node, s.Bcname))

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
		return err
	}
	var reply bson.A
	if cursor != nil {
		err = cursor.All(nil, &reply)
	}
	//fmt.Println("reply:", reply)

	//获取需要遍历的区块高度
	heights = make([]int64, len(reply))
	for i, v := range reply {
		heights[i] = v.(bson.D).Map()["_id"].(int64)
	}
	//fmt.Println("heights:", heights)

	//添加一个值,避免空指针异常
	heights = append(heights, block.Height)
	//找到缺少的区块
	lacks := findLacks(heights)

	//用个协程池,避免控制并发量
	defer ants.Release()
	wg := sync.WaitGroup{}
	p, _ := ants.NewPoolWithFunc(gosize, func(i interface{}) {
		func(height int64) {
			iblock, err := chain_server.GetBlockByHeight(s.Node, s.Bcname, height)
			if err != nil {
				log.Printf("get block by height failed, height: %d, error: %s", height, err)
				return
			}
			s.LackBlockChan <- iblock
		}(i.(int64))
		wg.Done()
	})
	defer p.Release()
	log.Println("缺少区块的数量：", len(lacks))
	for _, height := range lacks {
		//fmt.Println("start get lack block:", height)
		wg.Add(1)
		_ = p.Invoke(height)
	}
	wg.Wait()
	// 关闭管道
	// ?
	defer close(s.LackBlockChan)
	log.Println("get lack blocks finished")
	return nil
}
