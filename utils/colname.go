package utils

import "fmt"

//  返回collection名字
// count
func CountCol(node, bcname string) string {
	return fmt.Sprintf("count_%s_%s", node, bcname)
}

// block
func BlockCol(node, bcname string) string {
	return fmt.Sprintf("block_%s_%s", node, bcname)
}

// tx
func TxCol(node, bcname string) string {
	return fmt.Sprintf("tx_%s_%s", node, bcname)
}

// account
func AccountCol(node, bcname string) string {
	return fmt.Sprintf("account_%s_%s", node, bcname)
}
