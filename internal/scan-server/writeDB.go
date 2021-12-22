package scan_server

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/wxnacy/wgo/arrays"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"matrixchaindata/global"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/utils"
	"strconv"
	"sync"
)

// db 目前有4张表
// count 统计信息表
// block  区块表
// tx    交易信息表
// account 账号信息表
// v1.0版本是监听主链，v2.0 需要监听指定的链
// 现在设计是表是 在原理来的基础上加上链名作为后缀
// count_xxx
//  block_xxx
//  tx_xxx
//  account_xxx

var (
	gosize = 10
	counts *Count
	locker sync.Mutex
)

// todo 增加合约列表
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

//获取数据库中缺少的区块
func (w *WriteDB) HandleLackBlocks(node, bcname string) error {
	// 最新高度
	_, height, err := chain_server.GetUtxoTotalAndTrunkHeight(node, bcname)
	if err != nil {
		return err
	}
	// 获取缺少的区块
	return w.GetLackBlocks(&utils.InternalBlock{
		Height: height,
	}, node, bcname)
}

// 获取最新区块之前的block
// 读了两次数据库
func (w *WriteDB) GetLackBlocks(block *utils.InternalBlock, node, bcname string) error {
	log.Println("start get lack blocks")
	// 获取区块集合
	blockCol := w.MongoClient.Database.Collection(fmt.Sprintf("block_%s", bcname))

	//获取数据库中最后的区块高度
	sort := -1
	limit := int64(1)
	var heights []int64

again:
	{
		cursor, err := blockCol.Find(nil, bson.M{}, &options.FindOptions{
			Projection: bson.M{"_id": 1},
			Sort:       bson.M{"_id": sort},
			Limit:      &limit,
		})

		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}
		var reply bson.A
		if cursor != nil {
			err = cursor.All(nil, &reply)
		}
		//fmt.Println("reply:", reply)

		//获取需要遍历的区块高度
		heights = make([]int64, len(reply))
		for i, v := range reply {
			heights[i] = v.(bson.D).Map()["_id"].(int64)
		}
		//fmt.Println("heights:", heights)
	}

	//高度不匹配,找出缺少的区块高度，并获取区块
	if len(heights) == 1 && heights[0] != block.Height-1 {
		sort = 1         //顺序排列
		limit = int64(0) //获取所有区块
		goto again
	}

	//添加一个值,避免空指针异常
	heights = append(heights, block.Height)
	//找到缺少的区块
	lacks := findLacks(heights)

	//用个协程池,避免控制并发量
	defer ants.Release()
	wg := sync.WaitGroup{}
	p, _ := ants.NewPoolWithFunc(gosize, func(i interface{}) {
		func(height int64) {
			iblock, err := chain_server.GetBlockByHeight(node, bcname, height)
			if err != nil {
				log.Printf("get block by height failed, height: %d, error: %s", height, err)
				return
			}

			err = w.Save(utils.FromInternalBlockPB(iblock), node, bcname)
			if err != nil {
				log.Printf("save block to mongodb failed, height: %d, error: %s", height, err)
				return
			}
			//fmt.Println("succeed get lack block:", height)
		}(i.(int64))
		wg.Done()
	})
	defer p.Release()
	log.Println("缺少区块的数量：", len(lacks))
	for _, height := range lacks {
		//fmt.Println("start get lack block:", height)
		wg.Add(1)
		_ = p.Invoke(height)

		//未使用协程池
		//go func(height int64) {
		//	defer wg.Done()
		//	iblock, err := GetBlockByHeight(height)
		//	if err != nil {
		//		log.Println(err)
		//		return
		//	}
		//
		//	err = m.Save(utils.FromInternalBlockPB(iblock))
		//	if err != nil {
		//		log.Println(err)
		//		return
		//	}
		//	fmt.Println("succeed get lack block:", height)
		//
		//}(height)
	}

	wg.Wait()
	log.Println("get lack blocks finished")
	//fmt.Printf("running goroutines: %d\n", p.Running())
	return nil
}

// 找出缺少的区块
func findLacks(heights []int64) []int64 {
	log.Printf("mongodb's blocks size: %d", len(heights))

	if len(heights) == 0 {
		return nil
	}

	lacks := make([]int64, 0)

	var i int64 = 0
	for ; i < heights[len(heights)-1]; i++ {
		//不存在,记录该值
		index := arrays.ContainsInt(heights, i)
		if index == -1 {
			lacks = append(lacks, i)
			continue
		}
		//存在,剔除该值
		heights = append(heights[:index], heights[index+1:]...)
		//fmt.Println("heights:", heights)
	}
	//fmt.Println("lacks:", lacks)
	log.Printf("lack blocks size: %d", len(lacks))
	return lacks
}

// -----------------------------------------------------------------
//					         写入数据库
// -----------------------------------------------------------------
// 保存区块数据信息
func (w *WriteDB) Save(block *utils.InternalBlock, node, bcname string) error {
	// 多加一层判断，这个区块是否处理过了
	if w.IsHandle(block.Height, node, bcname) {
		log.Println("this block is handle", block.Height)
		return fmt.Errorf("height is handled, countine")
	}

	//存统计
	err := w.SaveCount(block, node, bcname)
	if err != nil {
		return err
	}

	//存交易
	err = w.SaveTx(block, node, bcname)
	if err != nil {
		return err
	}

	//存区块
	err = w.SaveBlock(block, node, bcname)
	if err != nil {
		return err
	}

	return nil
}

// 这个区块的数据是否处理过了？
func (w *WriteDB) IsHandle(block_id int64, node, bcname string) bool {
	blockCol := w.MongoClient.Database.Collection(fmt.Sprintf("block_%s", bcname))
	data := blockCol.FindOne(nil, bson.D{{"_id", block_id}})
	if data.Err() != nil {
		return false
	}
	return true
}

// 保存统计数据
func (w *WriteDB) SaveCount(block *utils.InternalBlock, node, bcname string) error {
	locker.Lock()
	defer locker.Unlock()

	// 总数统计集合
	countCol := w.MongoClient.Database.Collection(fmt.Sprintf("count_%s", bcname))
	// 账号统计集合
	accCol := w.MongoClient.Database.Collection(fmt.Sprintf("account_%s", bcname))

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
				//_, err := accCol.InsertOne(nil, bson.D{
				//	{"_id", txOutput.ToAddr},
				//	{"timestamp", tx.Timestamp},
				//})
				// by boxi
				_, err := accCol.UpdateOne(nil,
					bson.D{{"_id", txOutput.ToAddr}},
					bson.D{
						{"_id", txOutput.ToAddr},
						{"timestamp", tx.Timestamp}})
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

	txCol := w.MongoClient.Database.Collection(fmt.Sprintf("tx_%s", bcname))
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
		if tx.Desc == "1" { //投票奖励
			status = "vote_reward"
		} else if tx.Desc == "thaw" { //解冻
			status = "thaw"
		} else if tx.Desc == "award" { //出块奖励
			status = "block_reward"
		} else { //其他正常交易
			status = "normal"
		}

		_, err = txCol.ReplaceOne(nil,
			bson.M{"_id": tx.Txid},
			bson.D{
				{"_id", tx.Txid},
				{"status", status},
				{"height", height},
				{"tx", tx},
				//{"blockHeight", block.Height},
				//{"timestamp", tx.Timestamp},
				//{"initiator", tx.Initiator},
				////{"txInputs", tx.TxInputs},
				////{"txOutputs", tx.TxOutputs},
				//{"coinbase", tx.Coinbase},
				//{"voteCoinbase", tx.VoteCoinbase},
				{"state", state},
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

	//txCol := m.Database.Collection("tx")
	//_, err := txCol.InsertMany(m.ctx, sampleTxs)
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
		//{"blockid", block.Blockid},
		//{"proposer", block.Proposer},
		//{"transactions", txids},
		//{"txCount", block.TxCount},
		//{"preHash", block.PreHash},
		//{"inTrunk", block.InTrunk},
		//{"timestamp", block.Timestamp},
	}

	blockCol := w.MongoClient.Database.Collection(fmt.Sprintf("block_%s", bcname))
	_, err := blockCol.UpdateOne(
		nil,
		bson.D{{"_id", block.Height}},
		bson.D{{"$set", iblock}})
	return err
}
