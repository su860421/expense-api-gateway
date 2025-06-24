#!/bin/bash

# 測試運行腳本
# 用於運行 API Gateway 的所有測試

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函數
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
    
    if ! command -v go &> /dev/null; then
        print_error "Go 未安裝或不在 PATH 中"
        exit 1
    fi
    
    print_success "依賴檢查完成"
}

# 下載依賴
download_dependencies() {
    print_info "下載依賴..."
    
    go mod download
    go mod tidy
    
    print_success "依賴下載完成"
}

# 運行單元測試
run_unit_tests() {
    print_info "運行單元測試..."
    
    # 設置測試環境變數
    export GIN_MODE=test
    
    # 運行所有單元測試
    go test -v ./tests/unit/... -coverprofile=coverage_unit.out
    
    print_success "單元測試完成"
}

# 運行整合測試
run_integration_tests() {
    print_info "運行整合測試..."
    
    # 檢查服務是否運行
    if ! curl -s http://localhost:8088/health > /dev/null 2>&1; then
        print_warning "服務未運行，跳過整合測試"
        print_info "請先啟動服務: go run cmd/api/main.go"
        return
    fi
    
    # 運行整合測試
    go test -v ./tests/integration/... -coverprofile=coverage_integration.out
    
    print_success "整合測試完成"
}

# 運行部署測試
run_deployment_tests() {
    print_info "運行部署測試..."
    
    go test -v ./deployments/tests/... -coverprofile=coverage_deployment.out
    
    print_success "部署測試完成"
}

# 生成測試報告
generate_test_report() {
    print_info "生成測試報告..."
    
    # 合併覆蓋率報告
    if [ -f coverage_unit.out ] && [ -f coverage_integration.out ] && [ -f coverage_deployment.out ]; then
        go tool cover -func=coverage_unit.out > coverage_report.txt
        echo "" >> coverage_report.txt
        echo "=== Integration Tests ===" >> coverage_report.txt
        go tool cover -func=coverage_integration.out >> coverage_report.txt
        echo "" >> coverage_report.txt
        echo "=== Deployment Tests ===" >> coverage_report.txt
        go tool cover -func=coverage_deployment.out >> coverage_report.txt
        
        print_success "測試報告已生成: coverage_report.txt"
    else
        print_warning "部分測試未運行，無法生成完整報告"
    fi
}

# 運行基準測試
run_benchmark_tests() {
    print_info "運行基準測試..."
    
    go test -bench=. ./tests/unit/... -benchmem
    
    print_success "基準測試完成"
}

# 運行競態檢測
run_race_detection() {
    print_info "運行競態檢測..."
    
    go test -race ./tests/unit/...
    
    print_success "競態檢測完成"
}

# 清理測試文件
cleanup() {
    print_info "清理測試文件..."
    
    rm -f coverage_unit.out coverage_integration.out coverage_deployment.out
    
    print_success "清理完成"
}

# 顯示幫助信息
show_help() {
    echo "使用方法: $0 [選項]"
    echo ""
    echo "選項:"
    echo "  --unit              只運行單元測試"
    echo "  --integration       只運行整合測試"
    echo "  --deployment        只運行部署測試"
    echo "  --benchmark         運行基準測試"
    echo "  --race              運行競態檢測"
    echo "  --all               運行所有測試 (默認)"
    echo "  --cleanup           清理測試文件"
    echo "  --help, -h          顯示幫助信息"
    echo ""
    echo "環境變數:"
    echo "  GIN_MODE            Gin 模式 (test, debug, release)"
    echo "  TEST_TIMEOUT        測試超時時間"
}

# 主函數
main() {
    echo "========================================"
    echo "   智能報銷系統 API Gateway 測試"
    echo "========================================"
    echo ""
    
    # 解析命令行參數
    RUN_UNIT=false
    RUN_INTEGRATION=false
    RUN_DEPLOYMENT=false
    RUN_BENCHMARK=false
    RUN_RACE=false
    RUN_CLEANUP=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit)
                RUN_UNIT=true
                shift
                ;;
            --integration)
                RUN_INTEGRATION=true
                shift
                ;;
            --deployment)
                RUN_DEPLOYMENT=true
                shift
                ;;
            --benchmark)
                RUN_BENCHMARK=true
                shift
                ;;
            --race)
                RUN_RACE=true
                shift
                ;;
            --all)
                RUN_UNIT=true
                RUN_INTEGRATION=true
                RUN_DEPLOYMENT=true
                shift
                ;;
            --cleanup)
                RUN_CLEANUP=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                print_error "未知選項: $1"
                echo "使用 --help 查看幫助信息"
                exit 1
                ;;
        esac
    done
    
    # 如果沒有指定任何選項，運行所有測試
    if [ "$RUN_UNIT" = false ] && [ "$RUN_INTEGRATION" = false ] && [ "$RUN_DEPLOYMENT" = false ] && [ "$RUN_BENCHMARK" = false ] && [ "$RUN_RACE" = false ] && [ "$RUN_CLEANUP" = false ]; then
        RUN_UNIT=true
        RUN_INTEGRATION=true
        RUN_DEPLOYMENT=true
    fi
    
    # 檢查當前目錄
    if [ ! -f "go.mod" ]; then
        print_error "請在項目根目錄中運行此腳本"
        exit 1
    fi
    
    # 執行步驟
    check_dependencies
    download_dependencies
    
    if [ "$RUN_CLEANUP" = true ]; then
        cleanup
    fi
    
    if [ "$RUN_UNIT" = true ]; then
        run_unit_tests
    fi
    
    if [ "$RUN_INTEGRATION" = true ]; then
        run_integration_tests
    fi
    
    if [ "$RUN_DEPLOYMENT" = true ]; then
        run_deployment_tests
    fi
    
    if [ "$RUN_BENCHMARK" = true ]; then
        run_benchmark_tests
    fi
    
    if [ "$RUN_RACE" = true ]; then
        run_race_detection
    fi
    
    if [ "$RUN_UNIT" = true ] || [ "$RUN_INTEGRATION" = true ] || [ "$RUN_DEPLOYMENT" = true ]; then
        generate_test_report
    fi
    
    print_success "所有測試完成！"
}

# 執行主函數
main "$@" 