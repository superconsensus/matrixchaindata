package chain_server

import (
	"fmt"
	"google.golang.org/grpc"
)

// 创建grpc连接
func NewConnet(node string) *grpc.ClientConn {
	conn, err := grpc.Dial(node, grpc.WithInsecure())
	if err != nil {
		fmt.Println("unsubscribe failed, err msg:", err)
		return nil
	}
	return conn
}