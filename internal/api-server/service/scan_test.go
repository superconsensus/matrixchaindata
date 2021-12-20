package service

import (
	"fmt"
	"matrixchaindata/global"
	"matrixchaindata/internal/api-server/dao"
	"testing"
)

// 插入一条链记录
func TestServe_AddChain(t *testing.T) {

	node := "120.78.200.177:37101"
	bcname := "nft"
	Dao := GetDaoforScan()
	server := newServe(Dao)

	server.AddChain(node, bcname)
	//server.AddChain(node, bcname)
	server.AddChain(node, "nft")
	data, err := server.GetChainInfo(node, bcname)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(data)
	fmt.Println("------------")

	//result, _ := server.GetAllChainsInfo()
	//fmt.Println(result)
	//data, err := server.GetChainInfo(bcname, addr)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(data["node"])
	//fmt.Printf("%#v", data)
}

func GetDaoforScan() *dao.Dao {
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
