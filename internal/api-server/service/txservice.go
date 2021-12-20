package service

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"xuperdata/utils"
)

// 交易服务层
// 0 :全网， 1： 转入， 2： 转出
func (s *Serve) GetTxCount(bcname string, addr string, opt int64) (int64,error) {
	switch opt {
	case 0:
		// 查询链上的总交易数量
		return s.Dao.MongoClient.Collection(utils.TxCol(bcname)).CountDocuments(nil, bson.D{},options.Count())
	case 1:
		// 根据地址查询转入交易
		return s.Dao.MongoClient.Collection(utils.TxCol(bcname)).CountDocuments(
			nil,
			bson.D{
				{"tx.txoutputs.0.toaddr", addr},
			},
			options.Count())
	case 2:
		// 根据地址查询转出的交易
		return s.Dao.MongoClient.Collection(utils.TxCol(bcname)).CountDocuments(
		nil,
		bson.D{
			{"tx.txinputs.0.fromaddr", addr},
		},
		options.Count())
	default:
		return 0, fmt.Errorf("type error")
	}
}

// 交易列表
// 根据地址查询转入、转出数据
// opt 1 : 转入
// opt 2 : 转出
func (s *Serve) GetTxList(bcname, addr string, opt int64)([]bson.M, error)  {
	switch opt {
	case 1:
		cursur, err := s.Dao.MongoClient.Collection(utils.TxCol(bcname)).Find(
			nil,
			bson.D{
				{"tx.txoutputs.0.toaddr", addr},
			})
		if err != nil {
			return nil, err
		}
		var result []bson.M
		_ = cursur.All(nil, &result)
		return result, nil
	case 2:
		cursur, err :=  s.Dao.MongoClient.Collection(utils.TxCol(bcname)).Find(
			nil,
			bson.D{
				{"tx.txinputs.0.fromaddr", addr},
			})
		if err != nil {
			return nil, err
		}
		var result []bson.M
		_ = cursur.All(nil, &result)
		return result, nil
	default:
		return nil, fmt.Errorf("typer error")
	}
}
