package chain_server

import (
	"fmt"
	"testing"
)

func TestQueryBlockChains(t *testing.T) {
	node := "120.78.200.177:37101"
	//bcname := "nft"

	chains, _ := QueryBlockChains(node)
	for _, v := range chains {
		fmt.Println(v)
	}

	//conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	//if err != nil {
	//	return
	//}
	//defer conn.Close()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	//defer cancel()
	//
	//client := pb.NewXchainClient(conn)
	//client.Get
}
