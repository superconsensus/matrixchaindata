package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"xuperdata/internal/api-server/service"
	chain_server "xuperdata/internal/chain-server"
)

// 扫描控制器
type ScanController struct {}

// 添加一条链
// 节点 + 链名
// 不同网络，相同链名可以区分
// 相同网络，不同节点会造成重复数据
type AddChainReq struct {
	Bcname string `json:"bcname"`        // 链名
	Node string  `json:"node"`           // 节点
}
func (s *ScanController) AddChain(c *gin.Context)  {
	// 添加一条链
	params := &AddChainReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result": err,
		})
		return
	}

	//先校验，再记录数据
	// 查询节点上是否有这条链
	chains, err := chain_server.QueryBlockChains(params.Node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result": err,
		})
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
		fmt.Println(flag)
		// 不存在
		c.JSON(http.StatusBadRequest, gin.H{
			"result": "chain do not exist",
		})
		return
	}
	// 存在这条链
	// 记录下数据
	// 检查是否重复添加链，（数据库中是否有记录）
	result := service.NewSever().AddChain(params.Node, params.Bcname)
	if result == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result": "add chain error",
		})
		return
	}else if result == 1 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result": "chain is exist",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"rusult": "add chain success",
	})
	return
}

// 扫描请求参数
type ScanReq struct {
	Bcname string `json:"bcname"`
	Node string  `json:"node"`
}
// 启动扫描
func (s *ScanController) StartScan(c *gin.Context) {
	params := &ScanReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result": err,
		})
		return
	}
	// 节点 + 链名
	err =  service.NewSever().StartScanService(params.Node,params.Bcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result": fmt.Sprintf("start err: %v",err),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"result": "start success",
	})
	return
}

// 停止扫描
func (s *ScanController) StopScan(c *gin.Context){
	params := &ScanReq{}
	err := c.ShouldBindJSON(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result": err,
		})
		return
	}
	err =  service.NewSever().StopScanService( params.Node,params.Bcname,)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result": fmt.Sprintf("stop err: %v",err),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"result": "stop success",
	})
	return
}

// 获取已经添加的链的信息
func (s *ScanController) GetChainsInfo(c *gin.Context)  {
	allChains, err := service.NewSever().GetAllChainsInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result": err,
		})
		return
	}
	c.JSON(http.StatusOK,&allChains)
	return
}

// 获取正在扫描的链信息
func (s *ScanController) GetScanningChains(c *gin.Context) {
	scanningChains := service.NewSever().GetScanningChains()
	if len(scanningChains) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"result": "no scaning chain",
		})
		return
	}
	c.JSON(http.StatusOK, &scanningChains)
	return
}

