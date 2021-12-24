package service

import (
	"fmt"
	"matrixchaindata/global"
	"matrixchaindata/internal/api-server/dao"
	"testing"
)

func TestServe_GetTxCount(t *testing.T) {
	network := "02"
	bcname := "xuper"
	Dao := GetDaoforTx()
	server := newServe(Dao)

	data, err := server.GetChainInfo(network, bcname)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(data)

}

func GetDaoforTx() *dao.Dao {
	dbSource := "mongodb://admin:admin@192.168.199.128:27017"
	database := "boxi"
	//node := "120.79.69.94:37102"
	//bcname := "xuper"

	mgoClien, err := global.NewMongoClient(dbSource, database)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return dao.NewDao(mgoClien)

}
