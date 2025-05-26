FROM golang:1.24

WORKDIR /app

# 预安装依赖
COPY go.mod go.sum ./
RUN go mod download

# 设置测试环境
ENV CGO_ENABLED=0
ENV GO111MODULE=on

# 默认命令
CMD ["go", "test", "./..."] 