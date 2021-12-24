package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"matrixchaindata/internal/api-server/service"
	chain_server "matrixchaindata/internal/chain-server"
	"net/http"
	"strconv"
)

// db 目前有4张表
// count 统计信息表
// block  区块表
// tx    交易信息表
// account 账号信息表
// v1.0版本是监听主链，v2.0 需要监听指定的链
// 现在设计是表是 在原理来的基础上加上节点和链名作为后缀
// count_hash_xxx
//  block_hash_xxx
//  tx_hash_xxx
//  account_hash_xxx

// 交易控制器
type TxController struct{}

// 获取一笔交易信息
// bcname
// txid
type TxReq struct {
	Network string `json:"network"`
	Bcname  string `json:"bcname"`
	Txid    string `json:"txid"`
}

// 获取交易信息
// 直接调用链接口
func (t *TxController) GetTx(c *gin.Context) {
	// 获取参数
	params := TxReq{}
	err := c.ShouldBindJSON(&params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	// 参数校验
	if len(params.Txid) != 64 {
		fmt.Printf("error! txid must be 64 char, you input txid is: %s", params.Txid)
		c.JSON(http.StatusBadRequest, gin.H{"error": "txid error"})
		return
	}
	if params.Bcname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bcname is nil"})
		return
	}
	// 直接调用链服务查询
	chainInfo, err := service.NewSever().GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	tx, err := chain_server.GetTxByTxId(_node, params.Bcname, params.Txid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get the tx is failed"})
		return
	}
	c.JSON(http.StatusOK, tx)
	return
	// 返回
}

// 获取交易总量
// 默认是获取网的交易总量
type TxAmountReq struct {
	Network string `json:"network"`
	Bcname  string `json:"bcname"`
	Opt     string `json:"type"` // 0 :全网， 1： 转入， 2： 转出
	Addr    string `json:"addr"`
}

// 获取交易总数
func (t *TxController) GetTxAmount(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "bcname is nil"})
		return
	}
	opt, _ := strconv.ParseInt(params.Opt, 10, 64)
	// 调用service
	// 直接调用链服务查询
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	count, err := server.GetTxCount(_node, params.Bcname, params.Addr, opt)
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
	Network string `json:"network"`
	Bcname  string `json:"bcname"`
	Addr    string `json:"addr"`
	Opt     string `json:"opt"`
}

func (t *TxController) GetTxList(c *gin.Context) {
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
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	data, err := server.GetTxList(_node, params.Bcname, params.Addr, opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, &data)
	return
}

type GetContListReq struct {
	Network         string `json:"network"`
	Bcname          string `json:"bcname"`
	ContractAccount string `json:"contractaccount"`
	Address         string `json:"address"`
}

// 根据合约账号获取部署的合约
func (t *TxController) GetContractList(c *gin.Context) {
	params := &GetContListReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	// service
	chainInfo, err := service.NewSever().GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	// 调用链服务直接查询
	if params.Address != "" {
		data := chain_server.QueryAddressContracts(_node, params.Bcname, params.Address)
		c.JSON(http.StatusOK, data)
	} else if params.ContractAccount != "" {
		data := chain_server.QueryAccountContracts(_node, params.Bcname, params.ContractAccount)
		c.JSON(http.StatusOK, data)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "check params",
		})
	}
}

// 根据合约名字查出相关的交易
type ContractTxs struct {
	Network      string `json:"network"`
	Bcname       string `json:"bcname"`
	ContractName string `json:"contractname"`
}

func (t *TxController) GetContractTxs(c *gin.Context) {
	params := &ContractTxs{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	// 调用service
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		log.Println("info", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	data, err := server.GetContractTxs(_node, params.Bcname, params.ContractName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, &data)
}
