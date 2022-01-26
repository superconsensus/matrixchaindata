package response

// 定义统一的响应结果
var (
	// ok
	OK  = NewResponse(200, "ok")
	Err = NewResponse(500, "")

	// 服务级错误码
	ErrParam = NewResponse(10001, "参数有误")
)
