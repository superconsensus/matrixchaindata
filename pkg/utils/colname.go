package utils

import (
	"crypto/md5"
	"fmt"
	"io"
)

//  返回collection名字
// count
func CountCol(node, bcname string) string {
	return fmt.Sprintf("count_%s_%s", MD5(node, bcname), bcname)
}

// block
func BlockCol(node, bcname string) string {
	return fmt.Sprintf("block_%s_%s", MD5(node, bcname), bcname)
}

// tx
func TxCol(node, bcname string) string {
	return fmt.Sprintf("tx_%s_%s", MD5(node, bcname), bcname)
}

// account
func AccountCol(node, bcname string) string {
	return fmt.Sprintf("account_%s_%s", MD5(node, bcname), bcname)
}

// 对 node 和 bcname 做个md5
func MD5(node, bcname string) string {
	h := md5.New()
	io.WriteString(h, fmt.Sprintf("%s-%s", node, bcname))
	return fmt.Sprintf("%x", h.Sum(nil))
}
