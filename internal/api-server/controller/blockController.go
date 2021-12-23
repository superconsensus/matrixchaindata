package controller

import (
	"github.com/gin-gonic/gin"
	"matrixchaindata/internal/api-server/service"
	"net/http"
	"strconv"
)

// 区块控制器
type BlockController struct{}

// 获取区块信息
// 需要传入参数有链的名字
type GetBlockReq struct {
	Network string `json:"network"`
	Bcname  string `json:"bcname"`
	BlockId string `json:"blockid"`
	Height  string `json:"height"`
}

func (b *BlockController) GetBlock(c *gin.Context) {
	// 获取一个区块信息params
	params := &GetBlockReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err})
		return
	}
	// 根据网络类型和链名字获取数据
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	height, _ := strconv.ParseInt(params.Height, 10, 64)
	// 调用service获取数据
	// 直接去链上查询
	blockdata, err := server.GetBlockFormChain(_node, params.Bcname, params.BlockId, height)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "error",
		})
	}
	//c.JSON(http.StatusOK, gin.H{
	//	"data": data,
	//})
	c.JSON(http.StatusOK, blockdata)
}

// 根据链的名字获取区块高度
type GetBlockCount struct {
	Network string `json:"network"`
	Bcname  string `json:"bcname"`
}

func (b *BlockController) GetBlockCount(c *gin.Context) {
	params := &GetBlockCount{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err})
		return
	}

	// 调用server获取数据
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	height, err := server.GetBlockCountFromeChain(_node, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"blockCount": height,
	})
}

// 获取区块列表
type GetBlockList struct {
	Network     string `json:"network"`
	Bcname      string `json:"bcname"`
	BlockHeight string `json:"blockheight"`
	Num         string `json:"num"`
}

func (b *BlockController) GetBlockList(c *gin.Context) {
	params := &GetBlockList{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err})
		return
	}
	height, _ := strconv.ParseInt(params.BlockHeight, 10, 64)
	num, _ := strconv.ParseInt(params.Num, 10, 64)
	// 获取数据
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	_node := chainInfo["node"].(string)

	blocklist, err := server.GetBockekListFromChain(_node, params.Bcname, height, num)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err})
		return
	}
	c.JSON(http.StatusOK, blocklist)
}
