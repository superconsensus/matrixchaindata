package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/xuperchain/xuperchain/service/pb"
	"io"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"xuperdata/utils"
)

type Config struct {
	Type string `json:"type"`
	Args struct {
		Data string `json:"data"`
	} `json:"args"`
}

type TransactionEventRequest struct {
	Bcname      string `json:"bcname"`
	Initiator   string `json:"initiator"`
	AuthRequire string `json:"auth_require"`
	NeedContent bool   `json:"need_content"`
}

type BlockEventRequest struct {
	Bcname      string `json:"bcname"`
	Proposer    string `json:"proposer"`
	StartHeight int64  `json:"start_height"`
	EndHeight   int64  `json:"end_height"`
	NeedContent bool   `json:"need_content"`
}

type AccountEventRequest struct {
	Bcname      string `json:"bcname"`
	FromAddr    string `json:"from_addr"`
	ToAddr      string `json:"to_addr"`
	NeedContent bool   `json:"need_content"`
}

type PubsubClientCommand struct {
	//DescFile   string //事件订阅的描述文件
	//Command    string //订阅的操作：订阅或者取消订阅
	//EventID    string //事件的id
	DestIP     string //节点的ip
	cli *Cli
	cmd *cobra.Command

	filter      string
	oneline     bool
	skipEmptyTx bool
	nodeName    string  //节点名

	DataSource string //mongodb的数据源
	Database   string //mongodb的数据库
	HttpPort   int    //http服务的监听端口
	Gosize     int    //获取区块时的并发协程数
	Restore    bool   //是否清空数据库重新获取数据
	ShowBlock  bool   //是否在接收到区块时打印区块高度
}

// todo 添加重置数据库功能
func (cmd *PubsubClientCommand) addFlags() {
	//flag.StringVar(&cmd.DescFile, "f", "json/block.json", "arg file to subscribe an event")
	//flag.StringVar(&cmd.Command, "c", "subscribe", "option: subscribe|unsubscribe")
	//flag.StringVar(&cmd.EventID, "id", "000", "eventID to unsubscribe")
	//flag.StringVar(&cmd.DestIP, "h", "localhost:37101", "xchain node")
	cmd.cmd.Flags().StringVarP(&cmd.filter, "filter", "f", "{}", "filter options")
	cmd.cmd.Flags().BoolVarP(&cmd.oneline, "oneline", "", false, "whether print one event one line")
	cmd.cmd.Flags().BoolVarP(&cmd.skipEmptyTx, "skip-empty-tx", "", false, "whether print block with no tx matched")

	flag.StringVar(&cmd.DataSource, "s", "mongodb://admin:admin@0.0.0.0:27017", "mongodb data source")
	flag.StringVar(&cmd.Database, "b", "jy_chain", "mongodb database")
	flag.StringVar(&cmd.DestIP, "nodeIp", "localhost:37101", "node ip")
	flag.StringVar(&cmd.nodeName, "nodeName", "matrixchain", "node name")
	flag.IntVar(&cmd.HttpPort, "port", 8081, "port of http server")
	flag.IntVar(&cmd.Gosize, "gosize", 10, "goroutine size of sync block")
	flag.BoolVar(&cmd.Restore, "restore", false, "clean the database")
	flag.BoolVar(&cmd.ShowBlock, "show", false, "show received block's height")
	flag.Parse()
}

func (c *PubsubClientCommand) watch(ctx context.Context) error {

	//初始化数据库连接
	var err error
	mongoClient, err = NewMongoClient(c.DataSource, c.Database)
	if err != nil {
		log.Println("can't connecting to mongodb, err:", err)
		return nil
	}
	defer mongoClient.Close()

	fmt.Println(c.Restore)
	//清空数据库
	if c.Restore {
		err = mongoClient.Drop(nil)
		if err != nil {
			err = mongoClient.Database.Collection("count").Drop(nil)
			err = mongoClient.Database.Collection("account").Drop(nil)
			err = mongoClient.Database.Collection("tx").Drop(nil)
			err = mongoClient.Database.Collection("block").Drop(nil)
			if err != nil {
				log.Printf("clean the database failed, error: %v", err)
				return err
			}
		}
		fmt.Println("clean the database successed")
	}

	//data, err := ioutil.ReadFile(c.DescFile)
	//if err != nil {
	//	fmt.Println("subscribe failed, error:", err)
	//	return nil
	//}
	conn, err := grpc.Dial(c.DestIP, grpc.WithInsecure())
	if err != nil {
		fmt.Println("unsubscribe failed, err msg:", err)
		return nil
	}
	defer conn.Close()

	filter := &pb.BlockFilter{
		Bcname: c.nodeName,
	}
	bcname = c.nodeName
	node = c.DestIP
	err2 := json.Unmarshal([]byte(c.filter), filter)
	if err2 != nil {
		return err2
	}

	buf, _ := proto.Marshal(filter)
	request := &pb.SubscribeRequest{
		Type:   pb.SubscribeType_BLOCK,
		Filter: buf,
	}


	err = mongoClient.Init()     //获取数据库中缺少的区块
	if err != nil {
		log.Fatalf("get lack blocks failed, error: %s", err)
	}

	xclient := c.cli.EventClient()
	stream, err := xclient.Subscribe(ctx, request)
	if err != nil {
		return err
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var block pb.InternalBlock
		err = proto.Unmarshal(event.Payload, &block)
		if err != nil {
			return err
		}
		//if len(block.GetTxs()) == 0 && c.skipEmptyTx {
		//	continue
		//}
		if c.ShowBlock {
			fmt.Println("Recv block:", block.Height)
		}
		//存数据
		err = mongoClient.Save(utils.FromInternalBlockPB(&block))
		if err != nil {
			log.Printf("save block to mongodb failed, height: %d, error: %s", block.Height, err)
		}


		//c.printBlock(&block)
	}
	return nil
}

//func main() {
//
//	cli := NewCli()
//
//
//	c := new(PubsubClientCommand)
//	c.cli = cli
//
//	c.cmd = &cobra.Command{
//		Use:   "watch [options]",
//		Short: "watch block event",
//
//	}
//
//	//添加启动的命令参数，注释这句可以以上面的cmd对象来进行debug测试
//	c.addFlags()
//	conn, err := grpc.Dial(c.DestIP, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
//	if err != nil {
//		return
//	}
//	c.cli.xclient = pb.NewXchainClient(conn)
//	c.cli.eventClient = pb.NewEventServiceClient(conn)
//
//
//	port = c.HttpPort
//	go run() //开启http服务
//
//	ctx := context.TODO()
//	c.watch(ctx)
//
//	fmt.Println(c)
//}
