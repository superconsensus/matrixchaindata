package chain_server

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xuperchain/service/pb"
	"google.golang.org/grpc"
	"io"
	"log"
)

// 监听器
type Watcher struct {
	FilteredBlockChan <-chan *pb.InternalBlock
	Exit              chan<- struct{}
}

// 监听
// WatchBlockEvent new watcher for block event.
func WatchBlockEvent(bcname string, conn *grpc.ClientConn) (*Watcher, error) {
	// 创建监听器
	watcher := &Watcher{}
	// 区块过滤条件
	filter := &pb.BlockFilter{
		Bcname: bcname,
	}
	buf, _ := proto.Marshal(filter)
	request := &pb.SubscribeRequest{
		Type:   pb.SubscribeType_BLOCK,
		Filter: buf,
	}

	// 订阅时间
	xclient := pb.NewEventServiceClient(conn)
	stream, err := xclient.Subscribe(context.TODO(), request)
	if err != nil {
		return nil, err
	}

	// 创建管道，用于存放监听到的区块数据
	// 管道大小
	filteredBlockChan := make(chan *pb.InternalBlock, 50)
	exit := make(chan struct{})
	watcher.Exit = exit
	watcher.FilteredBlockChan = filteredBlockChan

	go func() {
		defer func() {
			close(filteredBlockChan)
			if err := stream.CloseSend(); err != nil {
				log.Printf("Unregister block event failed, close stream error: %v", err)
			} else {
				log.Printf("Unregister block event success...")
			}
		}()
		for {
			select {
			case <-exit:
				log.Println("stop watch")
				return
			default:
				event, err := stream.Recv()
				if err == io.EOF {
					return
				}
				if err != nil {
					log.Printf("Get block event err: %v", err)
					return
				}
				var block pb.InternalBlock
				err = proto.Unmarshal(event.Payload, &block)
				if err != nil {
					log.Printf("Get block event err: %v", err)
					return
				}
				//if &block == nil {
				//	continue
				//}
				filteredBlockChan <- &block
			}
		}
	}()
	return watcher, nil
}
