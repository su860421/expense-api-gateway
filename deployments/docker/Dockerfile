# 多階段構建 Dockerfile
FROM golang:1.21-alpine AS builder

# 設置工作目錄
WORKDIR /app

# 安裝必要的系統依賴
RUN apk add --no-cache git ca-certificates tzdata

# 複製 go mod 文件
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製源代碼
COPY . .

# 構建應用程式
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gateway \
    ./cmd/server

# 第二階段：運行環境
FROM alpine:latest

# 安裝ca-certificates和tzdata
RUN apk --no-cache add ca-certificates tzdata

# 創建非root用戶
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser

# 設置工作目錄
WORKDIR /app

# 從builder階段複製執行文件
COPY --from=builder /app/gateway .

# 複製配置文件
COPY --from=builder /app/configs ./configs

# 創建日誌目錄
RUN mkdir -p /var/log/gateway && \
    chown -R appuser:appgroup /app /var/log/gateway

# 切換到非root用戶
USER appuser

# 暴露端口
EXPOSE 8080 9090

# 健康檢查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 啟動應用程式
CMD ["./gateway"]
