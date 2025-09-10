# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

软件是作为vaultwarden的数据同步补充，主要参考vaultwarden-backup这个项目（https://github.com/ttionya/vaultwarden-backup）的思路。软件需要创建docker和vaultwarden的docker配合使用，使用的时候会将vaultwarden的data目录映射进来。

## Repository Structure

Currently, the repository contains only:
- `README.md`: Basic project description

## Development Status

✅ **项目基础结构已完成**
- Go 项目初始化，遵循最佳实践
- 使用 Echo 框架作为 Web 服务器
- Ent ORM 用于数据库操作（SQLite）
- Uber FX 依赖注入框架
- 完整的项目目录结构

✅ **核心功能已实现**
- 用户认证系统（JWT + Argon2）
- 设置向导功能
- WebDAV 和 S3 存储支持
- Vaultwarden 数据备份（支持压缩和加密）
- 自动同步调度系统
- 现代化前端界面（PicoCSS + HTMX）

✅ **开发基础设施**
- Docker 容器化支持
- Docker Compose 配置
- 单元测试覆盖
- Makefile 构建脚本
- GitHub Actions CI/CD
- 完整的文档

## Future Development

- 后端使用go语言开发，web框架使用echo框架，数据库使用ent orm，默认使用sqlite数据库。项目需要遵循golang项目的最佳实践，模块之间使用uber的fx进行依赖注入。每个功能都需要编写单元测试，单元测试测试通过以后再进行下一步开发。
- 后端需要实现简单的鉴权和参数设置的功能。在本地的配置文件中只能配置端口选项，当然也有默认值（8181），其他选项全在系统里面进行配置。软件启动的时候需要添加设置向导，引导用户完成配置。
- 后端需要支持添加webdav和s3作为存储，然后使用golang创建vaulwarden数据文件sqlite3（可以考虑支持其他数据库，前期只支持sqlite）的压缩包，支持压缩包密码设置，自动同步到多个存储，支持单向同步和由云上同步恢复。同步可以设置间隔，相关核心代码可以参考rclone的代码。
- 前端需要具备漂亮和现代的前端界面，使用https://picocss.com作为前端css框架，并且可以尝试使用htmlx框架，前端页面需要嵌入到服务端中，以单二进制运行。

## Notes

This CLAUDE.md file should be updated as the project structure and requirements become more defined during development.