#!/bin/bash

# Docker 构建验证脚本
# 用于验证 Docker 配置和构建过程

set -e

echo "=== Vaultwarden Syncer Docker 验证脚本 ==="

# 检查 Docker 是否可用
echo "1. 检查 Docker 环境..."
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装或不在 PATH 中"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "错误: Docker Compose 未安装或不在 PATH 中"
    exit 1
fi

echo "Docker 版本:"
docker --version
echo "Docker Compose 版本:"
docker-compose --version

# 检查必要文件
echo ""
echo "2. 检查必要文件..."
required_files=("Dockerfile" "docker-compose.yml" "go.mod" "go.sum")
for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✓ $file 存在"
    else
        echo "✗ $file 缺失"
        exit 1
    fi
done

# 验证 Docker Compose 配置
echo ""
echo "3. 验证 Docker Compose 配置..."
if docker-compose config > /dev/null 2>&1; then
    echo "✓ docker-compose.yml 配置有效"
else
    echo "✗ docker-compose.yml 配置无效"
    docker-compose config
    exit 1
fi

# 检查 Go 项目配置
echo ""
echo "4. 检查 Go 项目配置..."
if go mod verify > /dev/null 2>&1; then
    echo "✓ Go 模块验证通过"
else
    echo "⚠ Go 模块验证失败，这在构建时可能会有问题"
fi

# 尝试构建 Docker 镜像
echo ""
echo "5. 构建 Docker 镜像..."
if docker build -t vaultwarden-syncer-test . > /dev/null 2>&1; then
    echo "✓ Docker 镜像构建成功"
    
    # 检查镜像大小
    size=$(docker images vaultwarden-syncer-test --format "{{.Size}}")
    echo "  镜像大小: $size"
    
    # 清理测试镜像
    docker rmi vaultwarden-syncer-test > /dev/null 2>&1
else
    echo "✗ Docker 镜像构建失败"
    echo "构建日志:"
    docker build -t vaultwarden-syncer-test .
    exit 1
fi

# 检查健康检查命令
echo ""
echo "6. 验证健康检查命令..."
if command -v curl &> /dev/null; then
    echo "✓ curl 可用于健康检查"
else
    echo "⚠ curl 未找到，健康检查可能需要调整"
fi

# 检查配置文件
echo ""
echo "7. 检查配置文件..."
if [[ -f "config.yaml.example" ]]; then
    echo "✓ config.yaml.example 存在"
else
    echo "⚠ config.yaml.example 缺失"
fi

if [[ -f ".env.example" ]]; then
    echo "✓ .env.example 存在"
else
    echo "⚠ .env.example 缺失"
fi

# 建议的下一步
echo ""
echo "=== 验证完成 ==="
echo ""
echo "建议的下一步操作:"
echo "1. 复制配置文件："
echo "   cp config.yaml.example config.yaml"
echo "   cp .env.example .env"
echo ""
echo "2. 创建必要目录："
echo "   mkdir -p backups logs"
echo ""
echo "3. 启动服务："
echo "   docker-compose up -d"
echo ""
echo "4. 检查服务状态："
echo "   docker-compose ps"
echo "   curl http://localhost:8181/health"
echo ""
echo "5. 查看日志："
echo "   docker-compose logs -f vaultwarden-syncer"

echo ""
echo "更多详细信息请参考 docker-test.md 文件"