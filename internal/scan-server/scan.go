package scan_server

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/xuperchain/xuperchain/service/pb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"matrixchaindata/global"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/pkg/utils"
	"sync"
	"time"
)

var (
	// 协程池的容量
	gosize = 20
)

// 扫描器
type Scaner struct {
	// 链相关
	Node        string
	Bcname      string
	ChainClient *chain_server.ChainClient // 链客户端
	// 数据库
	DBWrite *WriteDB
	// 监听器用于获取数据
	Watcher *chain_server.Watcher
	// 接收数据管道（接受监听数据 + 接受缺少区块的数据 ）
	LackBlockChan chan *pb.InternalBlock
	// 退出方式，使用ctx
	Cannel context.CancelFunc
}

// 创建扫描器
func NewScanner(node, bcname string) (*Scaner, error) {
	// 创建链客户端连接
	client, err := chain_server.NewChainClien(node)
	if err != nil {
		return nil, fmt.Errorf("creat chain client fail")
	}

	//数据库处理
	// 传入的是全局db,不要在这里关闭。应该在mian中处理
	writeDB := NewWriterDB(global.GloMongodbClient)

	// 监听数据
	watcher, err := client.WatchBlockEvent(bcname)
	if err != nil {
		return nil, err
	}

	return &Scaner{
		Node:          node,
		Bcname:        bcname,
		ChainClient:   client,
		DBWrite:       writeDB,
		Watcher:       watcher,
		LackBlockChan: make(chan *pb.InternalBlock, 100),
	}, nil
}

// 停止扫描工作
// 关闭工作是这样的，连接不为空，则发送退出信号
// goroutin 接收到信息，则关闭grpc连接，在退出
// 监听数据很好控制
func (s *Scaner) Stop() {
	s.Cannel()
}

// 启动扫描工作
// 先扫描数据库中缺少的数据
// 然后再获取监听到的数据
func (s *Scaner) Start() error {
	// top goroutin
	// 第一层ctx, 以context.Background()为boot
	ctx1, cannel1 := context.WithCancel(context.Background())
	s.Cannel = cannel1

	go func() {
		// 这里开启协程池
		defer ants.Release()
		wg := sync.WaitGroup{}
		p, _ := ants.NewPoolWithFunc(gosize, func(i interface{}) {
			func(height int64) {
				iblock, err := s.ChainClient.GetBlockByHeight(s.Bcname, height)
				if err != nil {
					log.Printf("get block by height failed,bcname:%s, height: %d, error: %s", s.Bcname, height, err)
					return
				}
				s.LackBlockChan <- iblock
			}(i.(int64))
			wg.Done()
		})
		defer p.Release()

		heightChan, err := s.GetLackHeights(ctx1)
		if err != nil {
			log.Println("get lack height error", err)
			return
		}
		for height := range heightChan {
			wg.Add(1)
			_ = p.Invoke(height)
		}
		log.Println("get blocks finished", s.Node, s.Bcname)
		wg.Wait()
		close(s.LackBlockChan)
		log.Println("quit lack block gortution")
	}()

	go func() {
		defer func() {
			// 监听，如果出现错误关闭管道并退出
			// 通知监听goroutine退出，随后自己停止
			s.Watcher.Exit <- struct{}{}
			// 关闭
			//s.ChainClient.Close()
			return
		}()
		for {
			select {
			case <-ctx1.Done():
				// 推出管道是在监听退出信号的
				// 在退出的时候开始清理一些资源
				log.Println("stop scnner")
				// 1 通知监听器退出
				s.Watcher.Exit <- struct{}{}
				// 2 关闭grpc连接,(需要在里清理，需要关闭监听器之后关闭连接)
				log.Println("clear network source")
				_ = s.ChainClient.Close()
				//s.ChainClient.Close()
				return
			case block := <-s.Watcher.FilteredBlockChan:
				//log.Println("get data from watch chan", s.Bcname)
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
					log.Println("close LackBlockChan")
					s.LackBlockChan = nil
				}
			default:
				//没有监听到数据睡眠
				time.Sleep(1 * time.Microsecond)
			}
		}
	}()
	return nil
}

// ------------------------------
// 改进: 找到一个去链上获取，利用管道的特性
// 异步执行
// 使用context 改进
// 父gorution可以控制子gorutine结束, 子goroutin可以通知父goroutin完成任务
// ------------------------------
func (s *Scaner) GetLackHeights(ctx context.Context) (<-chan int64, error) {
	// 最新的区块高度
	_, H, err := s.ChainClient.GetUtxoTotalAndTrunkHeight(s.Bcname)
	if err != nil {
		return nil, err
	}
	log.Println("start get lack blocks", s.Node, s.Bcname)
	// 获取区块集合
	blockCol := s.DBWrite.MongoClient.Database.Collection(utils.BlockCol(s.Node, s.Bcname))
	//获取数据库中最后的区块高度
	limit := int64(0)
	var heights []int64

	cursor, err := blockCol.Find(nil, bson.M{}, &options.FindOptions{
		Projection: bson.M{"_id": 1},
		Sort:       bson.M{"_id": 1},
		Limit:      &limit,
	})

	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}
	var reply bson.A
	if cursor != nil {
		err = cursor.All(nil, &reply)
	}

	// 需要遍历数据的长度
	length := len(reply)
	//获取需要遍历的区块高度
	heights = make([]int64, length)
	for i, v := range reply {
		heights[i] = v.(bson.D).Map()["_id"].(int64)
	}

	// 处理，找出缺少区块的索引
	heightChan := make(chan int64, 20)
	if length == 0 {
		//第一次同步数据
		go func() {
			log.Println("get height by 1", s.Bcname)
			var i int64 = 1
			for {
				select {
				case <-ctx.Done():
					// 接收到上一级的结束任务通知
					close(heightChan)
					return
				default:
					heightChan <- i
					i++
					if i >= H {
						log.Println("遍历完成，高度获取完成,i , H", i, H)
						// 完成结束，通知父级goroutin
						close(heightChan)
						return
					}
				}
			}
		}()
	} else {
		go func() {
			log.Println("get height by 2", s.Bcname)
			var i int64 = 1
			for {
				select {
				case <-ctx.Done():
					log.Println("exit heightchan", s.Node, s.Bcname)
					//接收到上一级的结束任务通知
					close(heightChan)
					return
				default:
					if SearchInt64(heights, i) == -1 {
						heightChan <- i
					}
					i++
					if i >= H {
						// 完成结束，通知父级goroutin
						log.Println("遍历完成，高度获取完成,i , H", i, H)
						close(heightChan)
						return
					}
				}
			}
		}()
	}
	return heightChan, nil
}

// x 在 a 中的索引
func SearchInt64(a []int64, x int64) int64 {
	var index int64 = -1
	i, j := int64(0), int64(len(a))
	for i < j {
		// 中间
		middle := int64(uint64(i+j) >> 1)
		if a[middle] > x {
			j = middle
		} else if a[middle] < x {
			i = middle + 1
		} else {
			return middle
		}
	}
	return index
}
