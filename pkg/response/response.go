package response

// 统一封装api返回结果

// 响应结果
type Response struct {
	Code int         `json:"code"` // 错误码
	Msg  string      `json:"msg"`  // 信息提示
	Data interface{} `json:"data"` // 返回数据
}

// 构造函数
func NewResponse(code int, msg string) *Response {
	return &Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	}
}

// 添加数据
func (req *Response) WithData(data interface{}) *Response {
	return &Response{
		Code: req.Code,
		Msg:  req.Msg,
		Data: data,
	}
}

// 添加信息
func (req *Response) WithMsg(msg string) *Response {
	return &Response{
		Code: req.Code,
		Msg:  msg,
		Data: req.Data,
	}
}
