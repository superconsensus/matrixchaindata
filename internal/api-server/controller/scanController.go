package controller

import (
	"github.com/gin-gonic/gin"
	"matrixchaindata/internal/api-server/service"
	chain_server "matrixchaindata/internal/chain-server"
	"matrixchaindata/pkg/response"
	"net/http"
)

// 扫描控制器
type ScanController struct{}

// 添加一条链
// 网络 + 节点 + 链名
// 目前类型网络表示： 主网：01  测试网：02

// 扫描请求参数
type ScanReq struct {
	Network string `json:"network"` // 主网/测试网
	Node    string `json:"node"`    // 节点
	Bcname  string `json:"bcname"`  // 链名
}

// 添加一条链
// network, node, bcname
func (s *ScanController) AddChain(c *gin.Context) {
	// 添加一条链
	params := &ScanReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusOK, response.ErrParam)
		return
	}

	//先校验，再记录数据
	// 查询节点上是否有这条链
	chains, err := chain_server.QueryBlockChains(params.Node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}

	// 定义一个变量记录是否存在
	flag := false
	for _, v := range chains {
		if v == params.Bcname {
			flag = true
			break
		}
	}

	if !flag {
		// 不存在
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg("chain do not exist"))
		return
	}
	// 存在这条链
	// 记录下数据
	// 检查是否重复添加链，（数据库中是否有记录）
	result := service.NewSever().AddChain(params.Network, params.Node, params.Bcname)
	if result == 0 {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg("add chain error"))
		return
	} else if result == 1 {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg("chain is exist"))
		return
	}
	c.JSON(http.StatusOK, response.OK.WithData("add chain success"))
}

// 启动扫描
// network, bcname
func (s *ScanController) StartScan(c *gin.Context) {
	params := &ScanReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusOK, response.ErrParam)
		return
	}
	// 节点 + 链名
	err = service.NewSever().StartScanService(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK)
}

// 停止扫描
// network, bcname
func (s *ScanController) StopScan(c *gin.Context) {
	params := &ScanReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusOK, response.ErrParam)
		return
	}
	err = service.NewSever().StopScanService(params.Network, params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK)
}

// 获取已经添加的链的信息
func (s *ScanController) GetChainsInfo(c *gin.Context) {
	allChains, err := service.NewSever().GetAllChainsInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Err.WithMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK.WithData(allChains))
}

// 获取正在扫描的链信息
func (s *ScanController) GetScanningChains(c *gin.Context) {
	scanningChains := service.NewSever().GetScanningChains()
	if len(scanningChains) == 0 {
		c.JSON(http.StatusOK, response.OK.WithMsg("no scaning chain"))
		return
	}
	c.JSON(http.StatusOK, response.OK.WithData(scanningChains))
}
