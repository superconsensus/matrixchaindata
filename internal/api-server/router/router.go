package router

import (
	"github.com/gin-gonic/gin"
	"matrixchaindata/global"
	"matrixchaindata/internal/api-server/controller"
)

// api服务路由
// 利用gin重写

// 初始化路由
func InitRouter() *gin.Engine {
	// 路由引擎
	routerEngine := gin.New()
	gin.SetMode(global.Config.RunMode)
	// 添加日志中间件, 恢复中间件
	routerEngine.Use(gin.Logger())
	routerEngine.Use(gin.Recovery())

	/// count 统计信息
	// 如果可以链上查询尽量链上查询
	///// 区块路由组
	blockGroup := routerEngine.Group("/api/block")
	{
		block := &controller.BlockController{}
		// 获取区块
		blockGroup.POST("/get_block", block.GetBlock)
		//// 获取区块高度(当前链的区块数量)
		blockGroup.POST("/get_block_count", block.GetBlockCount)
		//// 获取区块列表
		blockGroup.POST("/get_block_list", block.GetBlockList)
	}

	/// 交易路由组
	txGroup := routerEngine.Group("/api/tx")
	{
		tx := &controller.TxController{}
		// 根据txid获取交易信息
		txGroup.POST("/get_tx", tx.GetTx)
		// 获取交易总量和列表
		txGroup.POST("/get_tx_amount", tx.GetTxAmount)
		//// 获取交易列表
		txGroup.POST("/get_tx_list", tx.GetTxList)

		// 合约相关
		// 根据合约账号/地址 获取部署的合约数量
		txGroup.POST("/get_contract_list", tx.GetContractList)
		// 根据合约名 查出相关的交易
		txGroup.POST("/get_contract_txs", tx.GetContractTxs)
	}

	// 扫描块程序相关
	///// 添加一个条链，方便浏览器切换
	scanGroup := routerEngine.Group("/api/scan")
	{
		scan := &controller.ScanController{}
		// 添加一条链
		scanGroup.POST("/add_chain", scan.AddChain)
		// 启动一条链
		scanGroup.POST("/start_scan", scan.StartScan)
		// 停止一条链
		scanGroup.POST("/stop_scan", scan.StopScan)
		// 获取已添加的链
		scanGroup.GET("/get_chains_info", scan.GetChainsInfo)
		// 获取正在扫面的链
		scanGroup.GET("/get_scanning_chain", scan.GetScanningChains)
	}

	// todo 增加
	/// account 账号信息相关

	return routerEngine
}
