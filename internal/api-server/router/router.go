package router

import (
	"github.com/gin-gonic/gin"
	"matrixchaindata/internal/api-server/controller"
)

// api服务路由
// 利用gin重写

// 初始化路由
func InitRouter(gin *gin.Engine) {

	/// count 统计信息

	///// 区块路由组
	blockGroup := gin.Group("/api/block")
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
	txGroup := gin.Group("/api/tx")
	{
		tx := &controller.TxController{}
		// 根据txid获取交易信息
		txGroup.POST("/get_tx", tx.GetTx)
		// 获取交易总量和列表
		txGroup.POST("/get_tx_amount", tx.GetTxAmount)
		//// 获取交易列表
		txGroup.POST("/get_tx_list", tx.GetTxList)
		// 合约相关
		//// 根据地址获取交易信息
		//txGroup.POST("/get_txlist_by_addr")
	}

	/// account 账号信息相关
	// 扫描块程序相关
	///// 添加一个条链，方便浏览器切换
	scanGroup := gin.Group("/api/scan")
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

}
