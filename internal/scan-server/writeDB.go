package scan_server

import (
	"encoding/json"
	"fmt"
	"github.com/wxnacy/wgo/arrays"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"matrixchaindata/pkg/sysinit"
	"matrixchaindata/pkg/utils"
	"strconv"
	"sync"
)

// db 目前有4张表
// count 统计信息表
// block  区块表
// tx    交易信息表
// account 账号信息表
var (
	// 统计数据，用于缓存
	counts *Count
	locker sync.Mutex
)

type Count struct {
	//ID        primitive.ObjectID `bson:"_id,omitempty"`
	TxCount   int64  `bson:"tx_count"`   //交易总数
	CoinCount string `bson:"coin_count"` //全网金额
	AccCount  int64  `bson:"acc_count"`  //账户总数
	Accounts  bson.A `bson:"accounts"`   //账户列表
	Contracts bson.A `bson:"contracts"`  //合约列表
}

// 写db结构体
type WriteDB struct {
	// mogodb的客户端
	MongoClient *sysinit.MongoClient
	// 节点
	Node string
	// 链名
	Bcname string
}

// 新建一个写数据的实例
func NewWriterDB(mongoclient *sysinit.MongoClient, node, bcname string) *WriteDB {
	return &WriteDB{
		MongoClient: mongoclient,
		Node:        node,
		Bcname:      bcname,
	}
}

// 谨慎，目前传入的是全局的db连接
func (w *WriteDB) Cloce() {
	w.MongoClient.Close()
}

// -----------------------------------------------------------------
//					         写入数据库
// -----------------------------------------------------------------
// 保存区块数据信息
func (w *WriteDB) Save(block *utils.InternalBlock, node, bcname string) error {

	// 多加一层判断，这个区块是否处理过了
	if w.IsHandle(block.Height) {
		log.Println("this block is handled", block.Height)
		return fmt.Errorf("height is handled, countine")
	}

	// 有一点需要注意的是block传过来的是指针，读数据就好了不要写。
	go func() {
		//存统计 （account, count表）
		err := w.SaveCount(block)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		//存交易 （tx表）
		err := w.SaveTx(block)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		//存区块 （block表）
		err := w.SaveBlock(block)
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

// 这个区块的数据是否处理过
func (w *WriteDB) IsHandle(block_height int64) bool {
	blockCol := w.MongoClient.Database.Collection(utils.BlockCol(w.Node, w.Bcname))
	data := blockCol.FindOne(nil, bson.D{{"_id", block_height}})
	if data.Err() != nil {
		// 没有记录,则没有处理过
		return false
	}
	return true
}

// 保存统计数据
func (w *WriteDB) SaveCount(block *utils.InternalBlock) error {
	locker.Lock()
	defer locker.Unlock()

	// 总数统计集合
	countCol := w.MongoClient.Database.Collection(utils.CountCol(w.Node, w.Bcname))
	// 账号统计集合
	accCol := w.MongoClient.Database.Collection(utils.AccountCol(w.Node, w.Bcname))

	//获取已有数据,缓存起来
	if counts == nil {
		counts = &Count{}

		//id必须有12个字节
		//获取统计数据
		err := countCol.FindOne(nil, bson.M{"_id": "chain_count"}).Decode(counts)
		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}

		//获取账户地址数据
		cursor, err := accCol.Find(nil, bson.M{})
		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}
		if cursor != nil {
			err = cursor.All(nil, &counts.Accounts)
		}
		//过滤key,减小体积
		for i, v := range counts.Accounts {
			counts.Accounts[i] = v.(bson.D).Map()["_id"]
		}
	}
	/// 统计数据
	//获取账户地址
	for _, tx := range block.Transactions {
		// 账户
		for _, txOutput := range tx.TxOutputs {
			//过滤矿工地址
			if txOutput.ToAddr == "$" {
				continue
			}
			//判断是否账户是否已存在
			i := arrays.Contains(counts.Accounts, txOutput.ToAddr)
			if i == -1 {
				//缓存账户
				counts.Accounts = append(counts.Accounts, txOutput.ToAddr)
				//写入数据库
				_, err := accCol.InsertOne(nil, bson.D{
					{"_id", txOutput.ToAddr},
					{"timestamp", tx.Timestamp},
				})
				if err != nil {
					return err
				}
			}
		}
		// 统计部署的合约
		if tx.ContractRequests != nil {
			for _, v := range tx.ContractRequests {
				// 判断合约名字是否存在
				i := arrays.Contains(counts.Contracts, v.ContractName)
				if i == -1 {
					// 缓存存起来
					counts.Contracts = append(counts.Contracts, v.ContractName)
				}
			}
		}
	}

	//统计账户总数
	counts.AccCount = int64(len(counts.Accounts))
	//统计交易总数
	counts.TxCount += int64(block.TxCount)

	up := true
	_, err := countCol.UpdateOne(nil,
		bson.M{"_id": "chain_count"},
		&bson.D{{"$set", bson.D{
			{"tx_count", counts.TxCount},
			{"coin_count", counts.CoinCount},
			{"acc_count", counts.AccCount},
			{"contract_count", counts.Contracts},
		}}},
		&options.UpdateOptions{Upsert: &up})

	return err
}

// 保存交易数据***
// -----------------------
// 核心：重点处理交易数据
// 数据格式问题
// todo 区分交易类型
// -----------------------
func (w *WriteDB) SaveTx(block *utils.InternalBlock) error {

	//索引 最新的交易
	//global.col.createIndex({"timestamp":-1}, {background: true})

	txCol := w.MongoClient.Database.Collection(utils.TxCol(w.Node, w.Bcname))
	up := true
	var err error

	//遍历交易,处理交易数据
	for _, tx := range block.Transactions {

		//交易类型---正常的转账
		status := "normal"
		//该交易是否成功
		state := "fail"
		//区块高度
		height := block.Height
		if tx.Blockid != nil {
			state = "success"
		}
		//截断一下,统一时间戳
		stringtime := strconv.FormatInt(tx.Timestamp, 10)
		if len(stringtime) > 13 {
			content := stringtime[0:13]
			tx.Timestamp, _ = strconv.ParseInt(content, 10, 64)
		}

		// 交易类型判断
		if tx.Desc == "1" { //投票奖励
			status = "vote_reward"
		} else if tx.Desc == "thaw" { //解冻
			status = "thaw"
		} else if tx.Desc == "award" { //出块奖励
			status = "block_reward"
		}

		// 合约调用判断
		if len(tx.ContractRequests) >= 1 {
			status = fmt.Sprintf("%s_contract", tx.ContractRequests[0].ContractName)
		}

		// 处理交易id显示
		txid, _ := tx.Txid.MarshalJSON()
		// 处理这笔交易
		txBytes, _ := json.MarshalIndent(tx, " ", " ")

		_, err = txCol.ReplaceOne(nil,
			bson.M{"_id": string(txid)},
			bson.D{
				{"_id", string(txid)},
				{"status", status},
				{"height", height},
				{"timestamp", tx.Timestamp},
				{"state", state},
				{"tx", string(txBytes)},
			},
			&options.ReplaceOptions{Upsert: &up})
	}
	return err
}

// 保存区块
func (w *WriteDB) SaveBlock(block *utils.InternalBlock) error {
	// 处理blockeid
	blockeid, _ := block.Blockid.MarshalJSON()

	iblock := bson.D{
		{"_id", block.Height},
		{"blockid", string(blockeid)},
		//{"proposer", block.Proposer},
		//{"transactions", txids},
		//{"txCount", block.TxCount},
		//{"preHash", block.PreHash},
		//{"inTrunk", block.InTrunk},
		{"timestamp", block.Timestamp},
	}

	blockCol := w.MongoClient.Database.Collection(utils.BlockCol(w.Node, w.Bcname))
	_, err := blockCol.InsertOne(nil, iblock)
	return err
}
