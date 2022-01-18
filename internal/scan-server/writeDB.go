package scan_server

import (
	"fmt"
	"github.com/wxnacy/wgo/arrays"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"matrixchaindata/global"
	"matrixchaindata/utils"
	"strconv"
	"sync"
)

// db 目前有4张表
// count 统计信息表
// block  区块表
// tx    交易信息表
// account 账号信息表
var (
	counts *Count
	locker sync.Mutex
)

type Count struct {
	//ID        primitive.ObjectID `bson:"_id,omitempty"`
	TxCount   int64  `bson:"tx_count"`   //交易总数
	CoinCount string `bson:"coin_count"` //全网金额
	AccCount  int64  `bson:"acc_count"`  //账户总数
	Accounts  bson.A `bson:"accounts"`   //账户列表
	Contracts bson.A `bson:"contracts"`  // 合约列表
}

// 写db结构体
type WriteDB struct {
	// mogodb的客户端
	MongoClient *global.MongoClient
}

// 新建一个写数据的实例
func NewWriterDB(mongoclient *global.MongoClient) *WriteDB {
	return &WriteDB{
		MongoClient: mongoclient,
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
	if w.IsHandle(block.Height, node, bcname) {
		log.Println("this block is handled", block.Height)
		return fmt.Errorf("height is handled, countine")
	}

	// todo 可以并发写的，他们不会操作同一张表
	// 有一点需要注意的是block传过来的是指针，读数据就好了不要写。
	//存统计 （account, count表）
	go func() {
		err := w.SaveCount(block, node, bcname)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		//存交易 （tx表）
		err := w.SaveTx(block, node, bcname)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		//存区块 （block表）
		err := w.SaveBlock(block, node, bcname)
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

// 这个区块的数据是否处理过
func (w *WriteDB) IsHandle(block_id int64, node, bcname string) bool {
	blockCol := w.MongoClient.Database.Collection(utils.BlockCol(node, bcname))
	data := blockCol.FindOne(nil, bson.D{{"_id", block_id}})
	if data.Err() != nil {
		// 没有记录
		return false
	}
	return true
}

// 保存统计数据
func (w *WriteDB) SaveCount(block *utils.InternalBlock, node, bcname string) error {
	locker.Lock()
	defer locker.Unlock()

	// 总数统计集合
	countCol := w.MongoClient.Database.Collection(utils.CountCol(node, bcname))
	// 账号统计集合
	accCol := w.MongoClient.Database.Collection(utils.AccountCol(node, bcname))

	//获取已有数据,缓存起来
	if counts == nil {
		counts = &Count{}

		//id必须有12个字节
		//获取统计数
		err := countCol.FindOne(nil, bson.M{"_id": "chain_count"}).Decode(counts)
		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}

		//获取账户地址
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
				// by boxi
				//_, err := accCol.UpdateOne(nil,
				//	bson.D{{"_id", txOutput.ToAddr}},
				//	bson.D{
				//		{"_id", txOutput.ToAddr},
				//		{"timestamp", tx.Timestamp}})
				if err != nil {
					return err
				}
			}
		}
		// 统计部署的合约合约
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

	// todo 修改获取金额的方式
	// 扫描旧的区块的时候，每次块都请求一次，链服务器压力大
	// io 过程漫长
	//统计全网金额
	//total, _, err := chain_server.GetUtxoTotalAndTrunkHeight(node, bcname)
	//if err != nil {
	//	log.Printf("get utxo total failed, height: %d, error: %s", block.Height, err)
	//} else {
	//	counts.CoinCount = total
	//}

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

// 保存交易数据
func (w *WriteDB) SaveTx(block *utils.InternalBlock, node, bcname string) error {

	//索引 最新的交易
	//global.col.createIndex({"timestamp":-1}, {background: true})

	txCol := w.MongoClient.Database.Collection(utils.TxCol(node, bcname))
	up := true
	var err error

	//遍历交易
	for _, tx := range block.Transactions {

		//交易类型
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
		// 合约调用
		if len(tx.ContractRequests) >= 1 {
			status = fmt.Sprintf("%s_contractt", tx.ContractRequests[0].ContractName)
		}

		_, err = txCol.ReplaceOne(nil,
			bson.M{"_id": tx.Txid},
			bson.D{
				{"_id", tx.Txid},
				{"status", status},
				{"height", height},
				{"timestamp", tx.Timestamp},
				{"state", state},
				{"tx", tx},
				//{"blockHeight", block.Height},
				//{"timestamp", tx.Timestamp},
				//{"initiator", tx.Initiator},
				////{"txInputs", tx.TxInputs},
				////{"txOutputs", tx.TxOutputs},
				//{"coinbase", tx.Coinbase},
				//{"voteCoinbase", tx.VoteCoinbase},
				//{"proposer", block.Proposer},
				//{"formaddress",formaddress},
				//{"toaddress",toaddress},
				//{"moduleName",moduleName},
				//{"contractName",contractName},
				//{"methodName",methodName},
				//{"args",args},
				//{"fromAmount",fromAmount},
				//{"toAmount",toAmount},
			},
			&options.ReplaceOptions{Upsert: &up})
	}
	return err
}

// 保存区块
func (w *WriteDB) SaveBlock(block *utils.InternalBlock, node, bcname string) error {

	txids := []bson.D{}
	for _, v := range block.Transactions {
		txids = append(txids, bson.D{
			{"$ref", "tx"},
			{"$id", v.Txid},
		})
	}

	iblock := bson.D{
		{"_id", block.Height},
		{"blockid", block.Blockid},
		{"proposer", block.Proposer},
		//{"transactions", txids},
		{"txCount", block.TxCount},
		{"preHash", block.PreHash},
		//{"inTrunk", block.InTrunk},
		{"timestamp", block.Timestamp},
	}

	blockCol := w.MongoClient.Database.Collection(utils.BlockCol(node, bcname))
	_, err := blockCol.InsertOne(nil, iblock)
	return err
}
