#!/bin/bash

# 设置错误时退出
set -e

# 启动 Redis 环境
echo "启动 Redis 环境..."
docker-compose up -d

# 等待 Redis 服务就绪
echo "等待 Redis 服务就绪..."
sleep 10  # 增加等待时间，确保集群完全初始化

# 运行单机模式测试
echo "运行单机模式测试..."
go test -v ./test/single/...

# 运行集群模式测试
echo "运行集群模式测试..."
go test -v ./test/cluster/...

# 清理环境
echo "清理测试环境..."
docker-compose down -v 