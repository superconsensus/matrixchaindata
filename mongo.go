package main

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/wxnacy/wgo/arrays"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"matrixchaindata/utils"
)

var (
	gosize      = 10         //苟柔婷数量
	mongoClient *MongoClient //全局的mongodb对象
)

type Count struct {
	//ID        primitive.ObjectID `bson:"_id,omitempty"`
	TxCount   int64  `bson:"tx_count"`   //交易总数
	CoinCount string `bson:"coin_count"` //全网金额
	AccCount  int64  `bson:"acc_count"`  //账户总数
	Accounts  bson.A `bson:"accounts"`   //账户列表
}

var counts *Count
var locker sync.Mutex

func (m *MongoClient) SaveCount(block *utils.InternalBlock) error {
	locker.Lock()
	defer locker.Unlock()

	countCol := m.Database.Collection("count")
	accCol := m.Database.Collection("account")

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
	}

	//统计账户总数
	counts.AccCount = int64(len(counts.Accounts))
	//统计交易总数
	counts.TxCount += int64(block.TxCount)
	//统计全网金额
	total, _, err := GetUtxoTotalAndTrunkHeight()
	if err != nil {
		log.Printf("get utxo total failed, height: %d, error: %s", block.Height, err)
	} else {
		counts.CoinCount = total
	}

	up := true
	_, err = countCol.UpdateOne(nil,
		bson.M{"_id": "chain_count"},
		&bson.D{{"$set", bson.D{
			{"tx_count", counts.TxCount},
			{"coin_count", counts.CoinCount},
			{"acc_count", counts.AccCount},
		}}},
		&options.UpdateOptions{Upsert: &up})

	return err
}

func (m *MongoClient) SaveTx(block *utils.InternalBlock) error {

	//索引 最新的交易
	//db.col.createIndex({"timestamp":-1}, {background: true})

	txCol := m.Database.Collection("tx")
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
		if tx.Blockid != "" {
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

		////获取formaddress
		//var formaddress string
		//if len(tx.TxInputs) <= 0 {
		//	formaddress = ""
		//}else {
		//	formaddress = tx.TxInputs[0].FromAddr
		//}
		////获取toaddress
		//var toaddress string
		////fromaccount
		//var fromAmount string
		//var toAmount string
		//for _ , address := range tx.TxOutputs{
		//	if address.ToAddr != formaddress && address.ToAddr != "$"{
		//		toaddress = address.ToAddr
		//		toAmount = address.Amount
		//	}
		//	if address.ToAddr == formaddress && address.ToAddr != "$"{
		//		fromAmount = address.Amount
		//	}
		//}
		//var moduleName string
		//var contractName string
		//var methodName string
		//var args map[string]string
		//if(len(tx.ContractRequests)>0) {
		//	moduleName = tx.ContractRequests[0].ModuleName
		//	contractName = tx.ContractRequests[0].ContractName
		//	methodName = tx.ContractRequests[0].MethodName
		//	args = tx.ContractRequests[0].Args
		//}

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

func (m *MongoClient) SaveBlock(block *utils.InternalBlock) error {

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
		{"transactions", txids},
		{"txCount", block.TxCount},
		{"preHash", block.PreHash},
		{"inTrunk", block.InTrunk},
		{"timestamp", block.Timestamp},
	}

	blockCol := m.Database.Collection("block")
	_, err := blockCol.InsertOne(nil, iblock)
	return err
}

func (m *MongoClient) Save(block *utils.InternalBlock) error {

	//存统计
	err := m.SaveCount(block)
	if err != nil {
		return err
	}

	//存交易
	err = m.SaveTx(block)
	if err != nil {
		return err
	}

	//存区块
	err = m.SaveBlock(block)
	if err != nil {
		return err
	}

	return nil
}

type MongoClient struct {
	*mongo.Client
	*mongo.Database
}

func NewMongoClient(dataSource, database string) (*MongoClient, error) {
	client, err := mongo.NewClient(options.Client().
		ApplyURI(dataSource).
		SetConnectTimeout(10 * time.Second))
	if err != nil {
		return nil, err
	}

	//ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	//defer client.Disconnect(ctx)

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	//databases, err := client.ListDatabaseNames(ctx, bson.M{})
	//if err != nil {
	//	return nil, err
	//}
	//fmt.Println(databases)

	return &MongoClient{client, client.Database(database)}, nil
}

func (m *MongoClient) Close() error {
	return m.Client.Disconnect(nil)
}

//找出缺少的区块
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

func (m *MongoClient) GetLackBlocks(block *utils.InternalBlock) error {
	log.Println("start get lack blocks")
	blockCol := m.Database.Collection("block")

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
			iblock, err := GetBlockByHeight(height)
			if err != nil {
				log.Printf("get block by height failed, height: %d, error: %s", height, err)
				return
			}

			err = m.Save(utils.FromInternalBlockPB(iblock))
			if err != nil {
				log.Printf("save block to mongodb failed, height: %d, error: %s", height, err)
				return
			}
			//fmt.Println("succeed get lack block:", height)
		}(i.(int64))
		wg.Done()
	})
	defer p.Release()
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

//获取数据库中缺少的区块
func (m *MongoClient) Init() error {
	_, height, err := GetUtxoTotalAndTrunkHeight()
	if err != nil {
		return err
	}
	return m.GetLackBlocks(&utils.InternalBlock{
		Height: height,
	})
}
