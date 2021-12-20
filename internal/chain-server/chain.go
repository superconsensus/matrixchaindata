package chain_server


/// 从链上拿到数据
import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/xuperchain/xuperchain/service/pb"
	"google.golang.org/grpc"
	//"net/http"
	"time"
)

//var (
//	node   = ":37101"
//	bcname = "xuper"
//)

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


	//查询单条链
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

// 查询链
// 返回
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