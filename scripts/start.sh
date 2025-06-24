#!/bin/bash

# 智能報銷系統 API Gateway 啟動腳本

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 輸出帶顏色的信息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 檢查依賴
check_dependencies() {
    print_info "檢查依賴..."
    
    # 檢查 Go
    if ! command -v go &> /dev/null; then
        print_error "Go 未安裝或不在 PATH 中"
        exit 1
    fi
    
    # 檢查 Go 版本
    GO_VERSION=$(go version | cut -d' ' -f3 | cut -d'o' -f2)
    REQUIRED_VERSION="1.21"
    if [[ $(echo "$GO_VERSION $REQUIRED_VERSION" | tr " " "\n" | sort -V | head -n1) != "$REQUIRED_VERSION" ]]; then
        print_error "需要 Go $REQUIRED_VERSION 或更高版本，當前版本: $GO_VERSION"
        exit 1
    fi
    
    print_success "Go $GO_VERSION 檢查通過"
}

# 設置環境變數
setup_environment() {
    print_info "設置環境變數..."
    
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
    
    # 從環境變數或使用默認值
    export SERVER_PORT=${SERVER_PORT:-8080}
    export SERVER_MODE=${SERVER_MODE:-development}
    export LOG_LEVEL=${LOG_LEVEL:-info}
    
    print_success "環境變數設置完成"
}

# 安裝依賴
install_dependencies() {
    print_info "安裝 Go 依賴..."
    
    if [ ! -f "go.mod" ]; then
        print_error "go.mod 文件不存在"
        exit 1
    fi
    
    go mod download
    go mod verify
    
    print_success "依賴安裝完成"
}

# 構建應用
build_application() {
    print_info "構建應用程式..."
    
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    go build \
        -ldflags="-w -s -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT" \
        -o bin/gateway \
        ./cmd/server
    
    print_success "應用程式構建完成"
}

# 運行測試
run_tests() {
    print_info "運行測試..."
    
    # 單元測試
    go test -v ./...
    
    # 競態檢測
    # go test -race ./...
    
    print_success "測試完成"
}

# 啟動應用
start_application() {
    print_info "啟動 API Gateway..."
    
    # 檢查端口是否被佔用
    if lsof -Pi :$SERVER_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "端口 $SERVER_PORT 已被佔用"
        print_info "嘗試終止佔用端口的進程..."
        PID=$(lsof -Pi :$SERVER_PORT -sTCP:LISTEN -t)
        kill -9 $PID 2>/dev/null || true
        sleep 2
    fi
    
    # 創建日誌目錄
    mkdir -p logs
    
    # 啟動應用
    print_success "API Gateway 正在啟動..."
    print_info "服務端口: $SERVER_PORT"
    print_info "運行模式: $SERVER_MODE"
    print_info "日誌級別: $LOG_LEVEL"
    print_info "配置文件: configs/config.yaml"
    
    exec ./bin/gateway
}

# 主函數
main() {
    echo "========================================"
    echo "   智能報銷系統 API Gateway 啟動器"
    echo "========================================"
    echo ""
    
    # 解析命令行參數
    SKIP_TESTS=false
    SKIP_BUILD=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --help|-h)
                echo "使用方法: $0 [選項]"
                echo ""
                echo "選項:"
                echo "  --skip-tests    跳過測試"
                echo "  --skip-build    跳過構建"
                echo "  --help, -h      顯示幫助信息"
                echo ""
                echo "環境變數:"
                echo "  SERVER_PORT     服務端口 (默認: 8080)"
                echo "  SERVER_MODE     運行模式 (默認: development)"
                echo "  LOG_LEVEL       日誌級別 (默認: info)"
                exit 0
                ;;
            *)
                print_error "未知選項: $1"
                echo "使用 --help 查看幫助信息"
                exit 1
                ;;
        esac
    done
    
    # 檢查當前目錄
    if [ ! -f "go.mod" ]; then
        print_error "請在項目根目錄中運行此腳本"
        exit 1
    fi
    
    # 執行步驟
    check_dependencies
    setup_environment
    install_dependencies
    
    if [ "$SKIP_TESTS" = false ]; then
        run_tests
    else
        print_warning "跳過測試"
    fi
    
    if [ "$SKIP_BUILD" = false ]; then
        build_application
    else
        print_warning "跳過構建"
        if [ ! -f "bin/gateway" ]; then
            print_error "可執行文件不存在，請先構建應用"
            exit 1
        fi
    fi
    
    start_application
}

# 信號處理
trap 'print_info "接收到中斷信號，正在關閉..."; exit 0' INT TERM

# 運行主函數
main "$@"
