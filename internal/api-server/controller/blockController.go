package controller

import (
	"github.com/gin-gonic/gin"
	"matrixchaindata/internal/api-server/service"
	"matrixchaindata/pkg/response"
	"net/http"
	"strconv"
)

// 区块控制器
type BlockController struct{}

// 获取区块信息
// 需要传入参数有链的名字
type BlockReq struct {
	Network string `json:"network"`
	Bcname  string `json:"bcname"`
	BlockId string `json:"blockid"`
	Height  string `json:"height"`
}

// 获取某个区块信息---要么根据高度也要么根据hash
func (b *BlockController) GetBlock(c *gin.Context) {
	// 获取一个区块信息params
	params := &BlockReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusOK, response.ErrParam)
		return
	}

	// 根据网络类型和链名字获取数据
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	_node := chainInfo["node"].(string)

	height, _ := strconv.ParseInt(params.Height, 10, 64)

	// 直接去链上查询
	blockdata, err := server.GetBlockFormChain(_node, params.Bcname, params.BlockId, height)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK.WithData(blockdata))
}

// 根据链的名字获取区块高度
// 传入 network, bcname
func (b *BlockController) GetBlockCount(c *gin.Context) {
	params := &BlockReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusOK, response.ErrParam)
		return
	}

	// 调用server获取数据
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	_node := chainInfo["node"].(string)

	height, err := server.GetBlockCountFromeChain(_node, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK.WithData(height))
}

// 获取区块列表
type GetBlockList struct {
	Network     string `json:"network"`
	Bcname      string `json:"bcname"`
	BlockHeight string `json:"blockheight"`
	Num         string `json:"num"`
}

// 指定高度之后的n条数据
func (b *BlockController) GetBlockList(c *gin.Context) {
	params := &GetBlockList{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusOK, response.ErrParam)
		return
	}
	height, _ := strconv.ParseInt(params.BlockHeight, 10, 64)
	num, _ := strconv.ParseInt(params.Num, 10, 64)

	// 获取链信息
	server := service.NewSever()
	chainInfo, err := server.GetChainInfo(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	_node := chainInfo["node"].(string)

	// 获取区块数据
	blocklist, err := server.GetBockekListFromChain(_node, params.Bcname, height, num)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK.WithData(blocklist))
}
