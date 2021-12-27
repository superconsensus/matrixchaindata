build:
	# 编译
	go build -o matrixdata main.go
run:
	# 运行服务，日志输出在nohup.out文件中
	nohup ./matrixdata >nohup.out 2>&1 &
stop:
	# 停止服务
	pkill matrixdata