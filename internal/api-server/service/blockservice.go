package service

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/utils"
)

// -----------------------------------------------------------------------------
// 							     获取区块信息
//								(1 链上  2 数据库)
// -----------------------------------------------------------------------------
// 根据区块hash, 或者是高度查早到区块信息
// 直接去链上拿
func (s *Serve) GetBlockFormChain(node, bcname, block_hash string, block_height int64) (*utils.InternalBlock, error) {

	// hash or id
	if block_hash == "" {
		// 根据高度获取
		// node 直接配置读取
		block, err := chain_server.GetBlockByHeight(node, bcname, block_height)
		if err != nil {
			return nil, err
		}
		return utils.FromInternalBlockPB(block), nil
	}
	// 根据hash获取
	block, err := chain_server.GetBlockById(node, bcname, block_hash)
	if err != nil {
		return nil, err
	}
	// 处理数据
	return utils.FromInternalBlockPB(block), nil
}

// 从数据库拿到数据
func (s *Serve) GetBlockFromDB(block_hash string, block_height int64, node, bcname string) (bson.M, error) {
	elem := bson.M{}
	err := s.Dao.MongoClient.Collection(utils.BlockCol(node, bcname)).FindOne(
		nil,
		bson.D{{"$or", bson.A{
			bson.D{{"blockid", block_hash}},
			bson.D{{"_id", block_height}},
		}},
		}).Decode(&elem)
	if err != nil {
		return nil, err
	}
	return elem, nil
}

// -----------------------------------------------------------------------------
// 							    获取区块数量
//                            (1 链上  2 数据库)
// -----------------------------------------------------------------------------
// 从链上拿数据
func (s *Serve) GetBlockCountFromeChain(node, bcname string) (int64, error) {
	_, height, err := chain_server.GetUtxoTotalAndTrunkHeight(node, bcname)
	if err != nil {
		return 0, err
	}
	return height, nil
}

// 重数据库拿到数据
// 链的区块高度（当前链出了多少个块）
func (s *Serve) GetBlockCountFromDB(node, bcname string) (int64, error) {

	//cursor, err := s.Dao.MongoClient.Collection(utils.BlockCol(bcname)).Find(nil,bson.D{})
	//if err != nil {
	//	return 0, err
	//}
	//result := []bson.M{}
	//cursor.All(nil,&result)
	return s.Dao.MongoClient.Collection(utils.BlockCol(node, bcname)).CountDocuments(nil, bson.D{}, options.Count())
	//return int64(len(result)), nil
}

// -----------------------------------------------------------------------------
// 							    获取区块列表
//                              ( 数据库)
// -----------------------------------------------------------------------------
// 从链上拿到数据
// 以高度作为开始下标获取指定条数的区块信息
func (s *Serve) GetBockekListFromChain(node, bcname string, height, num int64) ([]*utils.InternalBlock, error) {
	// 校验高度
	_, blockheight, err := chain_server.GetUtxoTotalAndTrunkHeight(node, bcname)
	if err != nil {
		return nil, fmt.Errorf("get block error")
	}
	if blockheight < height+num {
		return nil, fmt.Errorf(" height error,check it")
	}
	list, err := chain_server.GetBlockListByHeight(node, bcname, height, num)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 获取列表
// 获取区块列表
// 以高度作为开始下标获取指定条数的区块信息
func (s *Serve) GetBockekListFromDB(height int64, num int64, node, bcname string) ([]bson.M, error) {
	opts := options.Find()
	opts.SetSort(bson.D{{"_id", 1}})
	opts.SetLimit(num)

	cursor, err := s.Dao.MongoClient.Collection(utils.BlockCol(node, bcname)).Find(
		nil,
		bson.M{"_id": bson.M{"$gte": height}},
		opts)
	if err != nil {
		return nil, err
	}
	elems := []bson.M{}
	cursor.All(nil, &elems)
	return elems, nil
}
