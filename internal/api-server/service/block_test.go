package service

import (
	"fmt"
	"testing"
	"xuperdata/global"
	"xuperdata/internal/api-server/dao"
)

// 测试区块的读取
func TestServe_GetBlock(t *testing.T) {
	bcname := "xuper"
	Dao := GetDao()
	server := newServe(Dao)

	//height := 3403839
	//hash := "0e226c2af0ea75fef4b6e084b3cebb5447a9e4e5af17194085f13d26509d71eb"

	//doc, err := server.GetBlock(hash, int64(height), bcname)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(doc)
	//heght, _ := server.GetHeight(bcname)
	//fmt.Println("height:",heght)


	data, _ := server.GetBockekList(3403820, 5, bcname)
	for _, v := range data {
		fmt.Println(v)
	}

}

func GetDao()  *dao.Dao  {
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
