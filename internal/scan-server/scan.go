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
	// 接收数据管道（接受监听数据 + 接受缺少区块的数据 ）
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
		log.Println("handle old data")
		heightChain, exit, err := s.GetLackHeghts()
		if err != nil {
			log.Println(err)
		}
		err = s.GetBlocks(heightChain, exit)
		if err != nil {
			log.Println(exit)
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
				log.Println("get data from watch chan")
				// 处理监听器中数据
				err := s.DBWrite.Save(utils.FromInternalBlockPB(block), s.Node, s.Bcname)
				if err != nil {
					log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
				}
			case lackBlock, ok := <-s.LackBlockChan:
				//log.Println("get data from lack chan")
				// 处理缺少区块管道中的数据
				// 在完成的时候关闭管道
				// 防止不断读取关闭的管道
				if ok {
					err := s.DBWrite.Save(utils.FromInternalBlockPB(lackBlock), s.Node, s.Bcname)
					if err != nil {
						log.Printf("save block to mongodb failed, height: %d, error: %s", lackBlock.Height, err)
					}
				} else {
					log.Println("close LackBlockChan---")
					s.LackBlockChan = nil
				}
			default:
				//没有监听到数据睡眠
				time.Sleep(10 * time.Microsecond)
			}
		}
	}()

	return nil
}

// ------------------------------
// 改进: 找到一个去链上获取，利用管道的特性
// 异步执行
// ------------------------------
func (s *Scaner) GetLackHeghts() (<-chan int64, <-chan struct{}, error) {
	// 最新的区块高度
	_, H, err := chain_server.GetUtxoTotalAndTrunkHeight(s.Node, s.Bcname)
	if err != nil {
		return nil, nil, err
	}

	log.Println("start get lack blocks")
	// 获取区块集合
	blockCol := s.DBWrite.MongoClient.Database.Collection(utils.BlockCol(s.Node, s.Bcname))

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
		return nil, nil, err
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

	heightChan := make(chan int64, 20)
	exit := make(chan struct{})
	//找到缺少的区块
	// 性能杀手，数据量百万级直接龟速
	//lacks := findLacks(heights)
	// 改造下, 尝试用管道实现
	if len(heights) == 0 {
		// 第一次扫描，或者是清空了数据库
		// 需要同步的数据是 1 - H
		go func() {
			var i int64 = 1
			for ; i <= H; i++ {
				heightChan <- i
			}

			exit <- struct{}{}
		}()
	} else {
		//添加一个值,避免空指针异常
		//heights = append(heights, H)
		// 数据库中存在数据
		// heights[len(height)-1] ...  nowHeight
		go func() {
			var i int64 = 0
			for ; i < H; i++ {
				//不存在,记录该值
				index := arrays.ContainsInt(heights, i)
				if index == -1 {
					heightChan <- i
					continue
				}
				//存在,剔除该值
				// 消耗性能有可能是这部，切片频繁的变换
				//heights = append(heights[:index], heights[index+1:]...)
			}
			exit <- struct{}{}
		}()
	}
	return heightChan, exit, nil
}

func (s *Scaner) GetBlocks(heightChan <-chan int64, exit <-chan struct{}) error {
	client, err := chain_server.NewChainClien(s.Node)
	if err != nil {
		return err
	}
	defer client.Close()
	// 获取数据使用协程池
	//用个协程池,避免控制并发量
	defer ants.Release()
	wg := sync.WaitGroup{}
	p, _ := ants.NewPoolWithFunc(gosize, func(i interface{}) {
		func(height int64) {
			iblock, err := client.GetBlockByHeight(s.Bcname, height)
			if err != nil {
				log.Printf("get block by height failed, height: %d, error: %s", height, err)
				return
			}
			s.LackBlockChan <- iblock
		}(i.(int64))
		wg.Done()
	})
	defer p.Release()
	//for _, height := range lacks {
	//	wg.Add(1)
	//	_ = p.Invoke(height)
	//}

	// 监听高度管道的goroutine
	wg.Add(1)
	go func() {
		for {
			select {
			case <-exit:
				// 退出这个goroutin
				log.Println("exit heightchan")
				wg.Done()
			case height, ok := <-heightChan:
				if ok {
					wg.Add(1)
					_ = p.Invoke(height)
				} else {
					heightChan = nil
				}
			}
		}
	}()
	wg.Wait()
	log.Println("get blocks finished")
	close(s.LackBlockChan)
	return nil
}
