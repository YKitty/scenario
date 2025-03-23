.PHONY: build run clean test all

# 默认目标
all: build run

# 编译Go代码
build:
	@echo "编译程序..."
	@go build -o hashmap *.go

# 运行程序
run:
	@echo "运行程序..."
	@./hashmap

# 直接使用go run运行所有代码
go-run:
	@echo "直接运行所有Go文件..."
	@go run *.go

# 清理编译产物
clean:
	@echo "清理编译文件..."
	@rm -f hashmap

# 运行指定的并发测试
run-concurrent:
	@echo "选择要运行的并发测试:"
	@echo "1. 原始Channel实现"
	@echo "2. 互斥锁和条件变量实现"
	@echo "3. 三线程交替打印"
	@echo "4. 原子操作实现"
	@echo "5. 特定规则实现"
	@read -p "请输入选择 (1-5): " choice; \
	go run *.go <<< $$choice

# 帮助信息
help:
	@echo "使用说明:"
	@echo "  make build        - 编译Go代码"
	@echo "  make run          - 运行编译后的程序"
	@echo "  make go-run       - 直接使用go run运行代码"
	@echo "  make clean        - 清理编译产物"
	@echo "  make all          - 编译并运行程序 (默认)"
	@echo "  make run-concurrent - 运行并选择并发测试"
	@echo "  make help         - 显示帮助信息" 