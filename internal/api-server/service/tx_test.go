package service

import (
	"fmt"
	"testing"
)

func TestServe_GetTxCount(t *testing.T) {

	bcname := "xuper"
	Dao := GetDao()
	server := newServe(Dao)
	//"fromaddr" : "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
	count, _ := server.GetTxCount(bcname,"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",2)
	fmt.Println(count)

}