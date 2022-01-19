package chain_server

/// 从链上拿到数据
import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/xuperchain/xuperchain/service/pb"
	"google.golang.org/grpc"
	"matrixchaindata/pkg/utils"

	//"net/http"
	"time"
)

// 链客户端
type ChainClient struct {
	conn *grpc.ClientConn
	// 链客户端
	xc pb.XchainClient
	// 事件服务客户端
	esc pb.EventServiceClient
}

// 创建链客户端
func NewChainClien(node string) (*ChainClient, error) {
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}
	return &ChainClient{
		conn: conn,
		xc:   pb.NewXchainClient(conn),
		esc:  pb.NewEventServiceClient(conn),
	}, nil
}

// 关闭
func (c *ChainClient) Close() error {
	if c.xc != nil && c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// 获取区块by高度
func (c *ChainClient) GetBlockByHeight(bcname string, height int64) (*pb.InternalBlock, error) {

	blockHeightPB := &pb.BlockHeight{
		Bcname: bcname,
		Height: height,
	}
	reply, err := c.xc.GetBlockByHeight(context.TODO(), blockHeightPB)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, errors.New("GetBlockByHeight: the reply is null")
	}
	if reply.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return nil, errors.New("GetBlockByHeight: Header.Error is fail")
	}
	if reply.Block == nil {
		return nil, errors.New("GetBlockByHeight: the block is null")
	}
	return reply.Block, nil
}

// 获取utxo总量和高度
// args:
//      - node     节点地址
//      - bcname   链名字
// returns:
//      - total  utxo总量
//      - height 区块高度
func (c *ChainClient) GetUtxoTotalAndTrunkHeight(bcname string) (string, int64, error) {
	//查询单条链状态信息
	bcStatusPB := &pb.BCStatus{Bcname: bcname}
	bcStatus, err := c.xc.GetBlockChainStatus(context.TODO(), bcStatusPB)
	if err != nil {
		return "", -1, err
	}
	if bcStatus == nil {
		return "", -1, errors.New("GetBlockChainStatus: the chain is null")
	}
	if bcStatus.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return "-1", -1, errors.New("GetBlockChainStatus: Header.Error is fail")
	}

	total := bcStatus.UtxoMeta.UtxoTotal
	if err != nil {
		return "", -1, err
	}
	return total, bcStatus.Meta.TrunkHeight, nil
}

// --------------------------------------------------------------
//
// --------------------------------------------------------------

// 获取utxo总量和高度
// args:
//      - node     节点地址
//      - bcname   链名字
// returns:
//      - total  utxo总量
//      - height 区块高度
func GetUtxoTotalAndTrunkHeight(node, bcname string) (string, int64, error) {

	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return "", -1, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)

	//查询单条链状态信息
	bcStatusPB := &pb.BCStatus{Bcname: bcname}
	bcStatus, err := client.GetBlockChainStatus(ctx, bcStatusPB)
	if err != nil {
		return "", -1, err
	}
	if bcStatus == nil {
		return "", -1, errors.New("GetBlockChainStatus: the chain is null")
	}
	if bcStatus.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return "-1", -1, errors.New("GetBlockChainStatus: Header.Error is fail")
	}

	total := bcStatus.UtxoMeta.UtxoTotal
	if err != nil {
		return "", -1, err
	}
	return total, bcStatus.Meta.TrunkHeight, nil
}

// 根据高度获取区块
// args:
//      - node 节点地址
//      - bcname 链名字
//      - height 高度
// returns:
//      - 区块
//      - `error` : error
func GetBlockByHeight(node, bcname string, height int64) (*pb.InternalBlock, error) {

	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)
	blockHeightPB := &pb.BlockHeight{
		Bcname: bcname,
		Height: height,
	}

	reply, err := client.GetBlockByHeight(ctx, blockHeightPB)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, errors.New("GetBlockByHeight: the reply is null")
	}
	if reply.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return nil, errors.New("GetBlockByHeight: Header.Error is fail")
	}
	if reply.Block == nil {
		return nil, errors.New("GetBlockByHeight: the block is null")
	}
	return reply.Block, nil
}

func GetBlockListByHeight(node, bcname string, height, num int64) ([]*utils.InternalBlock, error) {
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)
	// 获取指定数量的区块信息 [)
	blocklist := make([]*utils.InternalBlock, 0)
	var i int64
	for i = 0; i < num; i++ {
		blockHeightPB := &pb.BlockHeight{
			Bcname: bcname,
			Height: height + i,
		}
		reply, err := client.GetBlockByHeight(ctx, blockHeightPB)
		if err != nil {
			continue
		}
		if reply == nil {
			continue
		}
		if reply.Header.Error != pb.XChainErrorEnum_SUCCESS {
			continue
		}
		if reply.Block == nil {
			continue
		}
		blocklist = append(blocklist, utils.FromInternalBlockPB(reply.Block))
	}
	if len(blocklist) == 0 {
		return nil, fmt.Errorf("get block error")
	}
	return blocklist, nil
}

// 根据blockid 获取区块id
// args:
//      - node 节点地址
//      - bcname 链名字
//      - blockid 高度
func GetBlockById(node, bcname, blockid string) (*pb.InternalBlock, error) {
	block_hash, err := hex.DecodeString(blockid)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)
	blockidPB := &pb.BlockID{
		Bcname:      bcname,
		Blockid:     block_hash,
		NeedContent: true,
	}
	block, err := client.GetBlock(ctx, blockidPB)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, errors.New("GetBlockByHeight: the reply is null")
	}
	if block.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return nil, errors.New("GetBlockByHeight: Header.Error is fail")
	}
	if block.Block == nil {
		return nil, errors.New("GetBlockByHeight: the block is null")
	}
	return block.Block, nil
}

// 根据交易id查询交易
// args:
//      - node 节点地址
//      - bcname 链名字
//      - height 高度
// return:
//      - 交易
//      - `error` error
func GetTxByTxId(node, bcname string, txid string) (*pb.Transaction, error) {

	rawTxid, err := hex.DecodeString(txid)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)
	txstatus := &pb.TxStatus{
		Bcname: bcname,
		Txid:   rawTxid,
	}

	reply, err := client.QueryTx(ctx, txstatus)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, errors.New("QueryTx: the reply is null")
	}

	if reply.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return nil, errors.New("QueryTx: Header.Error is fail")
	}
	if reply.Tx == nil {
		return nil, errors.New("QueryTx: the tx is null")
	}
	return reply.Tx, nil
}

// 查询链的数量
func QueryBlockChains(node string) ([]string, error) {
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)
	bcs, err := client.GetBlockChains(ctx, &pb.CommonIn{})
	if err != nil {
		return nil, err
	}

	if bcs.GetHeader().GetError() != pb.XChainErrorEnum_SUCCESS {
		return nil, errors.New(bcs.GetHeader().GetError().String())
	}

	return bcs.GetBlockchains(), nil
}

// 地址相关合约
func QueryAddressContracts(node, bcname, address string) map[string]*pb.ContractList {
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)

	req := &pb.AddressContractsRequest{
		Bcname:      bcname,
		Address:     address,
		NeedContent: true,
	}
	res, err := client.GetAddressContracts(ctx, req)
	if err != nil {
		return nil
	}
	return res.GetContracts()
}

// 合约账号相关合约
func QueryAccountContracts(node, bcname, account string) []*pb.ContractStatus {
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15000*time.Millisecond)
	defer cancel()

	client := pb.NewXchainClient(conn)

	req := &pb.GetAccountContractsRequest{
		Bcname:  bcname,
		Account: account,
	}

	res, err := client.GetAccountContracts(ctx, req)
	if err != nil {
		return nil
	}
	return res.GetContractsStatus()
}
