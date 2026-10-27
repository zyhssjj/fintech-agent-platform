# 本文件是后端容器构建模板，生产镜像可在此基础上补充迁移与健康检查。
FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/wfagent-server ./cmd/server

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /out/wfagent-server /app/wfagent-server
EXPOSE 8080
ENTRYPOINT ["/app/wfagent-server"]
