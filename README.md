# AI 智能報銷系統 - API Gateway

這是一個基於 **Go 1.24** 和 **Gin** 框架開發的企業級 API Gateway，專為 AI 智能報銷系統設計，採用現代清潔架構（Clean Architecture）原則，提供完整的微服務代理、認證授權、監控和安全功能。

## 🎯 專案定位

本 API Gateway 作為智能報銷系統的統一入口，專注於：
- **路由轉發** - 將請求轉發到對應的微服務
- **JWT 驗證** - 驗證 token 並提取用戶信息，但不存儲用戶數據
- **請求代理** - 作為微服務的統一入口點

## 🚀 功能特性

### ✅ 已完成功能

**🔐 認證授權**
- ✅ JWT Token 驗證中間件
- ✅ Token 過期檢查
- ✅ 用戶信息提取和轉發
- ✅ 角色基礎權限控制

**🛡️ 安全防護**
- ✅ CORS 跨域處理
- ✅ 請求/回應日志記錄
- ✅ 基礎安全配置
- ✅ 請求限流 (Rate Limiting)
- ✅ 請求體大小限制
- ✅ XSS 防護
- ✅ SQL 注入防護
- ✅ XSS 防護中間件完善
- ✅ SQL 注入防護中間件完善

**📊 監控記錄**
- ✅ 請求/回應日志
- ✅ API 調用統計
- ✅ 錯誤率監控
- ✅ 回應時間統計
- ✅ Prometheus 指標支援

**⚙️ 系統功能**
- ✅ 健康檢查
- ✅ 優雅關機
- ✅ 錯誤處理與回報
- ✅ 配置管理
- ✅ 配置熱重載
- ✅ 請求追蹤 (Request Tracing)

**🔌 服務整合**
- ✅ 服務發現
- ✅ 基礎代理轉發
- ✅ 動態路由解析器
- ✅ 輪詢 (Round Robin)
- ✅ 健康檢查
- ✅ 故障轉移

**📋 管理功能**
- ✅ 系統狀態檢查端點
- ✅ 維護模式切換
- ✅ 監控指標端點

### 🔄 進行中功能

**🔄 負載均衡**
- 🔄 加權輪詢實現
- 🔄 最少連接算法

**⚙️ 系統功能**
- 🔄 配置熱重載機制
- 🔄 請求追蹤系統實現

### ⏳ 待實現功能

**🔐 認證授權**
- ⏳ Token 刷新機制
- ⏳ API Key 認證支援
- ⏳ 基本認證 (Basic Auth) 支援

**🔄 負載均衡**
- ⏳ 高級負載均衡策略

**🔌 服務整合**
- ⏳ 請求/回應轉換
- ⏳ 協議轉換 (HTTP/gRPC)
- ⏳ 電路熔斷器 (Circuit Breaker)

**📋 管理功能**
- ⏳ API 版本管理
- ⏳ 路由規則管理
- ⏳ 黑白名單管理

## 🧪 測試覆蓋

本專案已涵蓋以下單元測試：
- JWT 認證中間件
- CORS 中間件
- 限流中間件
- SQL 注入防護中間件
- XSS 防護中間件
- 服務發現
- 代理服務
- 其他核心功能

執行所有單元測試：
```bash
go test ./tests/unit/... -v
```
或使用腳本：
```bash
./scripts/test.sh
```

## 🎯 架構分層說明

### 1. Presentation Layer (表現層)
- **handler/** - HTTP 處理器、路由、中間件
- **dto/** - 請求/響應數據結構
- **router/** - 路由配置

### 2. Business Layer (業務層)
- **service/** - 業務邏輯、用例實現
- **domain/** - 領域模型、實體、業務規則

### 3. Infrastructure Layer (基礎設施層)
- **infrastructure/** - 具體的 JWT、外部服務實現

### 4. Shared Layer (共享層)
- **pkg/** - 可重用的工具包
- **config/** - 配置管理

## 🔄 依賴方向

```
Handler → Service → Infrastructure
   ↓         ↓           ↓
  DTO    Domain      External Services
```

## 🚀 快速開始

### 環境要求
- **Go 1.24.2+** (已升級到最新穩定版本)
- Docker & Docker Compose (可選)
- Redis (用於限流和快取，可選)

### 本地開發

1. **克隆項目**
```bash
git clone <repository-url>
cd expense-api-gateway
```

2. **安裝依賴**
```bash
go mod download
go mod tidy
```

3. **配置文件**
```bash
# 配置文件已預設，可直接使用
# 如需自定義，可編輯 configs/config.yaml
```

4. **運行應用**

**使用啟動腳本 (推薦):**
```bash
# Linux/macOS
./scripts/start.sh

# Windows (PowerShell)
bash scripts/start.sh
```

**手動運行:**
```bash
go run cmd/api/main.go
```

5. **運行測試**
```bash
# 快速測試 (單元測試)
go test ./tests/unit/...

# 完整測試 (需要服務運行)
./scripts/test.sh
```

### Docker 部署

1. **使用 Docker Compose 啟動完整環境**
```bash
docker-compose up -d
```

2. **僅構建並運行 API Gateway**
```bash
docker build -t expense-api-gateway .
docker run -p 8080:8080 -p 9090:9090 expense-api-gateway
```

## 📋 API 文檔

### 健康檢查
```http
GET /health
```

### 系統狀態
```http
GET /api/v1/system/status
GET /api/v1/system/metrics
POST /api/v1/system/metrics/reset
```

### 服務發現
```http
GET /api/v1/services
GET /api/v1/services/{name}
POST /api/v1/services/{name}/register
DELETE /api/v1/services/{name}/{id}
```

### 代理轉發
```http
ANY /api/v1/proxy/{service-name}/{path}
```

### 管理端點
```http
GET /admin/config
POST /admin/config/reload
GET /admin/routes
POST /admin/maintenance
```

### 監控指標
```http
GET /metrics  # Prometheus 格式
```

## ⚙️ 配置說明

### 主配置文件 (`configs/config.yaml`)
- **服務器配置**: 端口、模式、超時設置
- **JWT配置**: Token 密鑰和過期時間
- **限流配置**: IP、用戶、API 限流規則
- **監控配置**: Prometheus 和指標設置
- **安全配置**: CORS、XSS、SQL 注入防護
- **日誌配置**: 級別、格式、輸出設置

### 微服務路由配置 (`configs/services.yaml`)
- **路由規則**: 路徑匹配、服務映射
- **服務配置**: 主機、端口、健康檢查
- **認證要求**: 是否需要認證、角色限制
- **超時設置**: 請求超時、最大請求體大小

## 🔧 開發指南

### 添加新的中間件
1. 在 `internal/middleware/` 目錄下創建新的中間件包
2. 實現 `gin.HandlerFunc` 接口
3. 在 `internal/router/router.go` 中註冊中間件

### 添加新的微服務
1. 在 `configs/services.yaml` 添加服務配置
2. 定義路由規則
3. 設置健康檢查端點
4. 重啟或重載配置

### 擴展認證機制
1. 在 `internal/domain/auth.go` 擴展認證模型
2. 在 `pkg/auth/` 實現認證邏輯
3. 更新 JWT Claims 結構
4. 修改認證中間件

## 📊 監控和觀測

### Prometheus 指標
- `http_requests_total`: HTTP 請求總數
- `http_request_errors_total`: HTTP 請求錯誤總數
- `http_request_duration_seconds`: HTTP 請求持續時間

### 日誌
結構化 JSON 日誌，包含：
- 請求詳情 (方法、路徑、狀態碼)
- 響應時間
- 客戶端 IP 和 User-Agent
- 錯誤信息

## 🧪 測試

### 測試架構

```
tests/
├── unit/                    # 單元測試
│   ├── middleware_auth_test.go      # JWT 認證中間件測試
│   ├── middleware_cors_test.go      # CORS 中間件測試
│   ├── middleware_ratelimit_test.go # 限流中間件測試
│   ├── service_discovery_test.go    # 服務發現測試
│   └── service_proxy_test.go        # 代理服務測試
├── integration/             # 整合測試
│   └── gateway_integration_test.go  # 端點整合測試
└── deployments/tests/       # 部署測試
    └── gateway_test.go      # 完整功能測試
```

### 測試覆蓋範圍

#### ✅ 已測試功能
- **JWT 認證中間件**: Token 驗證、角色權限、錯誤處理
- **CORS 中間件**: 預檢請求、實際請求、來源驗證
- **限流中間件**: IP 限流、用戶限流、API 限流
- **安全中間件**: XSS 防護、SQL 注入防護
- **服務發現**: 註冊、發現、監聽、健康檢查
- **代理服務**: 請求轉發、超時處理、標頭設置
- **基本端點**: 健康檢查、系統狀態、指標收集

#### 🔄 進行中測試
- **負載均衡**: 輪詢、加權輪詢、最少連接
- **監控服務**: 指標收集、統計分析

#### ⏳ 待測試功能
- **熔斷器**: 錯誤率檢測、自動恢復
- **配置熱重載**: 動態配置更新
- **請求追蹤**: 分散式追蹤、日誌關聯

### 運行測試

#### 使用測試腳本 (推薦)

**Linux/macOS:**
```bash
# 運行所有測試
./scripts/test.sh

# 只運行單元測試
./scripts/test.sh --unit

# 只運行整合測試
./scripts/test.sh --integration

# 只運行部署測試
./scripts/test.sh --deployment

# 運行基準測試
./scripts/test.sh --benchmark

# 運行競態檢測
./scripts/test.sh --race

# 清理測試文件
./scripts/test.sh --cleanup

# 查看幫助
./scripts/test.sh --help
```

**Windows (PowerShell):**
```powershell
# 運行所有測試
bash scripts/test.sh

# 只運行單元測試
bash scripts/test.sh --unit

# 只運行整合測試
bash scripts/test.sh --integration

# 只運行部署測試
bash scripts/test.sh --deployment

# 運行基準測試
bash scripts/test.sh --benchmark

# 運行競態檢測
bash scripts/test.sh --race

# 清理測試文件
bash scripts/test.sh --cleanup

# 查看幫助
bash scripts/test.sh --help
```

**注意:** Windows 用戶需要安裝 Git Bash 或 WSL 來運行 bash 腳本。

#### 手動運行測試
```bash
# 運行所有測試
go test ./...

# 運行單元測試
go test -v ./tests/unit/...

# 運行整合測試 (需要服務運行)
go test -v ./tests/integration/...

# 運行部署測試
go test -v ./deployments/tests/...

# 運行覆蓋率測試
go test -cover ./tests/unit/...

# 生成覆蓋率報告
go test -coverprofile=coverage.out ./tests/unit/...
go tool cover -html=coverage.out -o coverage.html

# 運行基準測試
go test -bench=. ./tests/unit/... -benchmem

# 運行競態檢測
go test -race ./tests/unit/...
```

### 測試環境設置

#### 單元測試
- 使用 `httptest` 模擬 HTTP 請求
- 使用 `gin.TestMode` 設置測試模式
- 不依賴外部服務

#### 整合測試
- 需要實際服務運行在 `localhost:8088`
- 測試真實的端點響應
- 驗證完整的請求-響應流程

#### 部署測試
- 使用完整的應用配置
- 測試路由設置和中間件鏈
- 驗證配置載入和服務初始化

### 測試最佳實踐

1. **測試隔離**: 每個測試都是獨立的，不依賴其他測試
2. **模擬外部依賴**: 使用 `httptest` 和 mock 對象
3. **測試覆蓋率**: 目標達到 80% 以上的代碼覆蓋率
4. **錯誤場景**: 測試正常和異常情況
5. **性能測試**: 使用基準測試驗證性能

### 持續整合

```yaml
# .github/workflows/test.yml 範例
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: go mod download
      - run: go test -v ./tests/unit/...
      - run: go test -cover ./tests/unit/...
      - run: go test -race ./tests/unit/...
```

## 📈 開發進度

### Phase 0: Go 升級 ✅
- ✅ 升級到 Go 1.24.2
- ✅ 更新 go.mod 和依賴檢查
- ✅ 驗證現有代碼相容性

### Phase 1: 核心功能實現 ✅
- ✅ JWT 認證中間件
- ✅ 動態路由解析器
- ✅ 代理服務重構
- ✅ 基礎限流中間件
- ✅ 服務發現機制
- ✅ 監控和日誌系統

### Phase 2: 測試和品質保證 ✅
- ✅ 單元測試框架
- ✅ 整合測試
- ✅ 部署測試
- ✅ 測試腳本和自動化
- ✅ 測試覆蓋率報告

### Phase 3: 進階功能 🔄
- ✅ 安全中間件完善 (XSS, SQL 注入)
- 🔄 負載均衡器優化
- 🔄 熔斷器模式實現
- ⏳ 配置熱重載
- ⏳ 請求追蹤系統

### Phase 4: 企業級功能 ⏳
- ⏳ 高級負載均衡策略
- ⏳ API 版本管理
- ⏳ 路由規則管理
- ⏳ 黑白名單管理
- ⏳ 分散式追蹤

## 📝 許可證

MIT License

## 🤝 貢獻

歡迎提交 Issue 和 Pull Request！

## 📞 支援

如有問題，請提交 Issue 或聯繫開發團隊。
