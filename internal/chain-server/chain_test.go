package chain_server

import (
	"encoding/json"
	"fmt"
	"matrixchaindata/pkg/utils"
	"testing"
)

func Test_GetBlockById(t *testing.T) {
	node := "120.78.200.177:37101"
	bcname := "xuper"
	blockid := "5367f4fcf29716bc72cf5ffa7f712fe7080baf2cf3704d77bf968cb3bee1b5ca"

	block, err := GetBlockById(node, bcname, blockid)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmtBlock := utils.FromInternalBlockPB(block)

	// 测试SaveTx
	output, err := json.Marshal(fmtBlock.Transactions[0])
	if err != nil {
		fmt.Println(err)
	}

	//output, err := json.MarshalIndent(fmtBlock, "", "  ")
	//if err != nil {
	//	fmt.Println(err)
	//}
	fmt.Println(string(output))
}
