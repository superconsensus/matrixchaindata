package utils

import "fmt"

//  返回collection名字
// count
func CountCol(bcname string) string {
	return fmt.Sprintf("count_%s", bcname)
}
// block
func BlockCol(bcname string) string {
	return fmt.Sprintf("block_%s",bcname)
}
// tx
func TxCol(bcname string) string {
	return fmt.Sprintf("tx_%s",bcname)
}

// account
func AccountCol(bcname string) string {
	return fmt.Sprintf("account_%s",bcname)
}

