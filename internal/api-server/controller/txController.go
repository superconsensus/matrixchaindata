package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"xuperdata/internal/api-server/service"
	chain_server "xuperdata/internal/chain-server"
	"xuperdata/settings"
)

// 交易控制器
type TxController struct {}

// 获取一笔交易信息
// bcname
// txid
type TxReq struct {
	Bcname string `json:"bcname"`
	Txid string `json:"txid"`
}
// 获取交易信息
// 直接调用链接口
func (t *TxController) GetTx(c *gin.Context){
	// 获取参数
	valid := TxReq{}
	err := c.ShouldBindJSON(&valid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	fmt.Printf("%#v", valid)
	// 参数校验
	if len(valid.Txid) != 64 {
		fmt.Printf("error! txid must be 64 char, you input txid is: %s", valid.Txid)
		c.JSON(http.StatusBadRequest,gin.H{"error":"txid error"})
		return
	}
	if valid.Bcname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error":"bcname is nil"})
		return
	}
	// 直接调用链服务查询
	tx, err := chain_server.GetTxByTxId(settings.Setting.Node, valid.Bcname, valid.Txid)
	if err != nil {
		c.JSON(http.StatusInternalServerError,gin.H{"error":"get the tx is failed"})
		return
	}
	c.JSON(http.StatusOK, tx)
	return
	// 返回
}


// 获取交易总量
// 默认是获取网的交易总量
type TxAmountReq struct {
	Bcname string `json:"bcname"`
	Opt string `json:"type"`    // 0 :全网， 1： 转入， 2： 转出
	Addr string `json:"addr"`
}

// 获取交易总数
func (t *TxController) GetTxAmount(c *gin.Context)  {
	// 获取参数
	params := TxAmountReq{}
	err := c.ShouldBindJSON(&params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	// 校验参数
	if params.Bcname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error":"bcname is nil"})
		return
	}
	opt, _ := strconv.ParseInt(params.Opt, 10, 64)
	// 调用service
	count, err := service.NewSever().GetTxCount(params.Bcname, params.Addr, opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// 获取交易列表
// opt 1 转入， 2 转出
type GetList struct {
	Bcname string `json:"bcname"`
	Addr string `json:"addr"`
	Opt string `json:"opt"`
}
func (t *TxController) GetTxList(c *gin.Context){
	params := &GetList{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	opt, _ := strconv.ParseInt(params.Opt, 10, 64)
	// 调用service
	data, err := service.NewSever().GetTxList(params.Bcname, params.Addr, opt)
	if err != nil{
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK,&data)
	return
}
