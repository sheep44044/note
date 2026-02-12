# Gnote

> 一个现代化、AI 驱动的笔记与知识库应用。（并非 前端中看不中用）

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/backend-Go%201.22+-00ADD8.svg)
![React](https://img.shields.io/badge/frontend-React%2018+-61DAFB.svg)

## 概要
是一个AI 驱动的现代化全栈知识库系统。它不仅仅是一个简单的 Markdown 编辑器，更是一个融合了语义搜索、自动内容生成与社区互动的智能笔记服务。

## 文档

[apifox 接口文档](https://app.apifox.com/project/7830692)


##  快速开始 (Quick Start)

### 1. 前置环境 (Prerequisites)

请确保你的开发环境已安装以下工具：

- [Go](https://go.dev/) (1.21 或更高版本)
- [Node.js](https://nodejs.org/) (18 或更高版本)
- [Docker](https://www.docker.com/) & Docker Compose

### 2. 初始化配置
克隆项目后，首先初始化配置文件：

```bash
make init
```

注意: 请务必编辑 .env 文件，填写你的 VOLC_ENGINE_KEY (火山引擎 API Key) 和其他必要的配置。

### 3. 启动基础设施

使用 Docker Compose 一键启动所有依赖服务 (MySQL, Redis, Qdrant, MinIO, RabbitMQ)：

```Bash
make infra
```

### 4. 运行应用

启动后端服务:在 http://localhost:8080

```Bash
make backend
```
启动前端服务:在 http://localhost:5173

```Bash
make frontend
```
## 项目总览

Gnote 是一个基于 Gin + React 的现代化全栈 AI 笔记/知识库系统。它实现了从笔记录入、自动向量化索引、AI 辅助阅读（摘要/标题）到自然语言语义检索的完整闭环，采用容器化部署方案。

- 代码语言与版本：Go 1.22+ (Backend) / TypeScript + React 18 (Frontend)

- 存储与中间件：MySQL 8.0 (业务数据)、Redis (缓存/限流)、RabbitMQ (异步解耦)、MinIO (对象存储)、Qdrant (向量数据库)

- AI 与 搜索：集成 VolcEngine (火山引擎) 大模型与 Embedding 能力，结合 Qdrant 实现 Hybrid Search (关键词+语义检索)

- 异步任务流：基于 RabbitMQ 实现 Write-Behind 模式，异步处理耗时的 AI 摘要生成与向量索引构建，保障写入性能

- 权限与安全：JWT (无状态认证) + Middleware 级鉴权 + Redis 分布式限流 (Rate Limiting)

- 可观测性：集成 Jaeger 实现全链路追踪 (Tracing)，配合 Zap 进行结构化日志管理

## 项目结构
Plaintext
.
├── cmd/                # 程序入口
│   └── main.go         # 主程序启动文件
├── config/             # 配置加载模块
├── internal/           # 内部业务逻辑 (核心代码)
│   ├── infra/          # 基础设施层 (MySQL, Redis, RabbitMQ, MinIO, Qdrant, AI)
│   ├── middleware/     # Gin 中间件 (JWT Auth, Logger, RateLimit, Tracer)
│   ├── models/         # 数据库模型定义 (GORM Structs)
│   ├── note/           # 笔记核心业务 (CRUD, Search, Community, Reaction)
│   ├── svc/            # 服务上下文 (Service Context, 依赖注入容器)
│   ├── tag/            # 标签管理业务
│   ├── user/           # 用户体系业务 (Login, Register, Profile, Follow)
│   ├── utils/          # 通用工具库 (JWT, Response, Logger, Helpers)
│   └── validators/     # 请求参数校验逻辑 (Binding & Validation)
├── web/                # React 前端项目 (Vite + Shadcn/UI)
├── docker-compose.yaml # 容器编排配置 (定义 MySQL, Redis 等服务)
├── Dockerfile          # 后端容器镜像构建文件
├── Makefile            # 项目自动化管理工具 (init, infra, run...)
├── go.mod              # Go 依赖定义文件
└── .env                # 环境变量配置文件


## 核心功能

笔记管理: Markdown 编辑、图片上传、私密/公开状态切换、置顶归档。

智能增强: 笔记保存时 可触发 AI 生成摘要和标题。

互动社区: 用户关注流 (Feed)、emoji回复 (Reaction)、收藏夹。

安全鉴权: 基于 JWT 的无状态认证 和 限流防刷机制。

混合检索: 支持 SQL 关键词搜索与 Vector 语义搜索并存。

