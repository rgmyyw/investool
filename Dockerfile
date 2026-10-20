# 多阶段构建：源码编译，替代原依赖 goreleaser dist 产物的单行 Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath \
    -ldflags "-s -w -X github.com/axiaoxin-com/investool/version.Version=$(date -u +%y%m%d%H%M)" \
    -o investool .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /srv/investool
RUN mkdir -p data
COPY --from=builder /app/investool .
COPY --from=builder /app/config.toml .
EXPOSE 4869 4870
# data 目录挂卷持久化基金/行业等 JSON 数据，重启不重新全量拉取
VOLUME ["/srv/investool/data"]
ENTRYPOINT ["./investool", "webserver", "-c", "./config.toml"]
