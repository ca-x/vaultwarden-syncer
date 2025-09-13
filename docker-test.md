# Docker 构建和运行测试指南

本文档说明如何构建和运行 vaultwarden-syncer 的 Docker 容器。

## 前置要求

- Docker Engine 20.0+
- Docker Compose 2.0+

## 构建测试

### 1. 构建单个容器

```bash
# 构建 vaultwarden-syncer 镜像
docker build -t vaultwarden-syncer .

# 验证镜像构建成功
docker images | grep vaultwarden-syncer
```

### 2. 使用 Docker Compose 构建

```bash
# 构建所有服务
docker-compose build

# 验证构建
docker-compose config
```

## 运行测试

### 1. 准备配置文件

```bash
# 复制配置文件示例
cp config.yaml.example config.yaml
cp .env.example .env

# 编辑配置文件（根据实际需要调整）
# config.yaml - 应用配置
# .env - 环境变量
```

### 2. 创建必要的目录

```bash
mkdir -p backups
mkdir -p logs
```

### 3. 运行服务

#### 仅运行 vaultwarden-syncer

```bash
docker-compose up vaultwarden-syncer
```

#### 运行完整栈（包括 vaultwarden）

```bash
docker-compose up
```

#### 运行时包含 MinIO（用于 S3 测试）

```bash
docker-compose --profile minio up
```

#### 运行时包含 nginx 代理

```bash
docker-compose --profile proxy up
```

### 4. 健康检查

```bash
# 检查容器状态
docker-compose ps

# 检查应用健康状态
curl http://localhost:8181/health

# 检查 vaultwarden 健康状态
curl http://localhost:8080/alive
```

## 测试用例

### 1. 基本功能测试

1. **访问 vaultwarden-syncer 管理界面**
   ```bash
   curl http://localhost:8181
   ```

2. **访问 vaultwarden 界面**
   ```bash
   curl http://localhost:8080
   ```

3. **测试设置向导**
   - 打开浏览器访问 `http://localhost:8181`
   - 完成初始设置向导
   - 配置存储提供者（S3 或 WebDAV）

### 2. 存储测试

#### S3 存储测试（使用 MinIO）

```bash
# 启动包含 MinIO 的服务
docker-compose --profile minio up -d

# 访问 MinIO 管理界面
# http://localhost:9001
# 用户名: minioadmin
# 密码: minioadmin123

# 创建存储桶并配置
# 在 vaultwarden-syncer 中配置 S3 存储
```

#### WebDAV 存储测试

```bash
# 需要外部 WebDAV 服务器
# 在 vaultwarden-syncer 中配置 WebDAV 存储
```

### 3. 备份和同步测试

1. **手动触发备份**
   - 通过 Web 界面触发备份
   - 检查备份文件是否上传到配置的存储

2. **自动同步测试**
   - 配置同步间隔
   - 等待自动同步触发
   - 验证备份文件

### 4. 数据恢复测试

1. **从备份恢复**
   - 下载备份文件
   - 测试恢复功能
   - 验证数据完整性

## 日志查看

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs vaultwarden-syncer
docker-compose logs vaultwarden

# 实时跟踪日志
docker-compose logs -f vaultwarden-syncer
```

## 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除卷（注意：会删除数据）
docker-compose down -v

# 停止并删除镜像
docker-compose down --rmi all
```

## 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 修改 docker-compose.yml 中的端口映射
   ports:
     - "8182:8181"  # 改为其他端口
   ```

2. **权限问题**
   ```bash
   # 确保目录权限正确
   sudo chown -R 1001:1001 ./data ./logs ./backups
   ```

3. **构建失败**
   ```bash
   # 清理构建缓存
   docker-compose build --no-cache
   
   # 检查 Go 模块
   docker run --rm -v $(pwd):/app -w /app golang:1.22 go mod verify
   ```

4. **健康检查失败**
   ```bash
   # 检查容器内部
   docker-compose exec vaultwarden-syncer /bin/sh
   
   # 检查应用进程
   docker-compose exec vaultwarden-syncer ps aux
   ```

## 性能优化

### 生产环境建议

1. **使用多阶段构建**（已实现）
   - 减少最终镜像大小
   - 仅包含运行时依赖

2. **资源限制**
   ```yaml
   # 在 docker-compose.yml 中添加
   deploy:
     resources:
       limits:
         memory: 512M
         cpus: '0.5'
   ```

3. **日志轮转**
   ```yaml
   # 配置日志轮转
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```

## 安全考虑

1. **环境变量**
   - 不要在 git 中提交 `.env` 文件
   - 使用强密码和随机令牌

2. **网络安全**
   - 在生产环境中使用 HTTPS
   - 配置防火墙规则

3. **数据加密**
   - 启用备份加密
   - 使用强密码保护备份文件

这个测试指南提供了完整的 Docker 构建、运行和测试流程。