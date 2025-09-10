# Vaultwarden Syncer

一个用于 Vaultwarden 数据同步的工具，作为 Vaultwarden 的数据同步补充，主要参考 [vaultwarden-backup](https://github.com/ttionya/vaultwarden-backup) 项目的思路。

## 功能特性

- 🔐 **安全认证** - 基于 JWT 的用户认证系统
- 📦 **数据备份** - 支持 Vaultwarden 数据的压缩备份
- 🔒 **加密保护** - 支持备份文件密码加密
- ☁️ **多存储支持** - 支持 WebDAV 和 S3 兼容的存储后端
- ⏰ **定时同步** - 可配置的自动同步间隔
- 🌐 **现代界面** - 使用 PicoCSS 和 HTMX 的现代化 Web 界面
- 🐳 **容器化** - 完整的 Docker 支持
- 🔧 **设置向导** - 首次运行时的引导配置

## 快速开始

### 使用 Docker Compose (推荐)

1. 克隆仓库：
```bash
git clone https://github.com/username/vaultwarden-syncer.git
cd vaultwarden-syncer
```

2. 复制配置文件：
```bash
cp config.yaml.example config.yaml
```

3. 编辑配置文件，设置你的存储后端和同步选项。

4. 启动服务：
```bash
docker-compose up -d
```

5. 访问 `http://localhost:8181` 完成初始设置。

### 手动构建

1. 确保已安装 Go 1.21+
2. 克隆仓库并进入目录
3. 生成 Ent 代码：
```bash
go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema
```
4. 构建项目：
```bash
go build -o vaultwarden-syncer ./cmd/server
```
5. 运行：
```bash
./vaultwarden-syncer
```

## 配置

### 基本配置

配置文件 `config.yaml` 的基本结构：

```yaml
server:
  port: 8181

database:
  driver: sqlite3
  dsn: "./data/syncer.db"

auth:
  jwt_secret: "your-secret-key-here"

storage:
  webdav: []
  s3: []

sync:
  interval: 3600          # 同步间隔（秒）
  compression_level: 6    # 压缩级别 (1-9)
  password: ""            # 备份文件密码（可选）
```

### WebDAV 存储配置

```yaml
storage:
  webdav:
    - name: "Nextcloud"
      url: "https://cloud.example.com/remote.php/webdav/"
      username: "your-username"
      password: "your-password"
```

### S3 存储配置

```yaml
storage:
  s3:
    - name: "AWS S3"
      endpoint: ""  # 留空使用 AWS S3
      access_key_id: "your-access-key"
      secret_access_key: "your-secret-key"
      region: "us-east-1"
      bucket: "your-bucket"
    - name: "MinIO"
      endpoint: "https://minio.example.com"
      access_key_id: "your-access-key"
      secret_access_key: "your-secret-key"
      region: "us-east-1"
      bucket: "vaultwarden-backups"
```

## API 端点

- `GET /` - 主页面（需要认证）
- `GET /setup` - 初始设置页面
- `POST /api/setup` - 完成初始设置
- `GET /login` - 登录页面
- `POST /api/login` - 处理登录
- `GET /health` - 健康检查

## 开发

### 项目结构

```
├── cmd/server/          # 应用程序入口点
├── internal/
│   ├── auth/           # 认证服务
│   ├── backup/         # 备份功能
│   ├── config/         # 配置管理
│   ├── database/       # 数据库连接
│   ├── handler/        # HTTP 处理程序
│   ├── scheduler/      # 定时任务调度
│   ├── server/         # HTTP 服务器
│   ├── service/        # 业务逻辑服务
│   ├── setup/          # 初始设置
│   ├── storage/        # 存储提供商
│   └── sync/           # 同步服务
├── ent/schema/         # 数据库模式定义
├── Dockerfile          # Docker 镜像构建
├── docker-compose.yml  # Docker Compose 配置
└── config.yaml.example # 配置文件示例
```

### 运行测试

```bash
go test ./...
```

### 技术栈

- **后端**: Go + Echo 框架
- **数据库**: SQLite (通过 Ent ORM)
- **依赖注入**: Uber FX
- **前端**: PicoCSS + HTMX
- **认证**: JWT + Argon2 密码哈希
- **存储**: WebDAV + S3 兼容
- **压缩加密**: ZIP + AES-256-GCM

## 许可证

MIT License

## 贡献

欢迎提交 Pull Request 和 Issue！

## 致谢

本项目参考了 [vaultwarden-backup](https://github.com/ttionya/vaultwarden-backup) 项目的设计思路。
