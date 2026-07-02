# 链动2+1分销系统

[![CI](https://github.com/yang-lei-cmd/chain2plus1/actions/workflows/ci.yml/badge.svg)](https://github.com/yang-lei-cmd/chain2plus1/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

基于 **Go + Gin + GORM** 的后端 + **React + TypeScript + Vite** 前端的全栈分销系统。

## 技术栈

| 层 | 技术 | 说明 |
|----|------|------|
| **后端** | Go 1.26 / Gin / GORM | RESTful API，多模块分层 |
| **数据库** | MySQL 5.7+ | 关系型数据库，GORM AutoMigrate |
| **前端** | React 19 / TypeScript / Vite | 移动端优先 SPA |
| **实时通知** | WebSocket (gorilla/websocket) | 指数退避重连，心跳保活 |
| **文档** | Swagger / OpenAPI | `GET /swagger/*any` |
| **部署** | Docker Compose | MySQL + API 一键启动 |
| **CI** | GitHub Actions | 自动 Vet → Build → Test → Docker |

## 功能模块

### Phase 1-3: 核心业务
- ✅ 用户注册/登录 (JWT 鉴权)
- ✅ 邀请码绑定 + 关系链
- ✅ 链动分润引擎 (5级分润: 10%→8%→5%→3%→2%)
- ✅ 自动解锁机制 (2个直推下线触发)
- ✅ 商品下单 + 余额管理

### Phase 4: 提现与后台
- ✅ 提现申请 (最低100元, 1%手续费)
- ✅ 管理员审核 (批准/拒绝)
- ✅ 排行榜 (收益/团队/充值)
- ✅ 管理后台 (用户/订单/商品/供应商)

### Phase 5: 支付与灵活用工
- ✅ 第三方支付 (微信/支付宝 Mock)
- ✅ 自由职业者注册/审核
- ✅ 任务发布/分配/提交/审核
- ✅ 工时记录 + 薪资结算 + 评分系统

### Phase 6-7: 实时通知与看板
- ✅ WebSocket 实时通知
- ✅ 数据分析看板 (收益趋势/用户增长/订单统计)

### 生产加固
- ✅ 速率限制 (100请求/10秒/IP)
- ✅ 安全头 (XSS/Clickjack/HSTS)
- ✅ 优雅关闭 (SIGTERM → 10s drain)
- ✅ Docker 多阶段构建 (~15MB)
- ✅ Swagger API 文档

## 快速开始

### 本地开发

```bash
# 1. 确保 MySQL 运行并创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS chain2plus1 CHARACTER SET utf8mb4"

# 2. 启动后端
cd chain2plus1
go run ./cmd/main.go
# → http://localhost:8080

# 3. 启动前端 (新终端)
cd chain2plus1/frontend
npm install
npm run dev
# → http://localhost:5173 (自动代理 API 到 8080)
```

### Docker 部署

```bash
docker compose up -d
# → http://localhost:8080
```

### 测试

```bash
# 全部测试
go test -count=1 ./...

# 仅 P0 (核心业务流)
go test -v -count=1 ./internal/handler/ -run "Test_Register|Test_Login|Test_CreateOrder|Test_ListProfits|Test_ApplyWithdraw|Test_ApproveWithdraw|Test_FullRegistration"

# 仅 P1 (支付+灵活用工)
go test -v -count=1 ./internal/handler/ -run "Test_Payment|Test_Freelancer|Test_E2E"

# 仅 P2 (链引擎+并发+边界)
go test -v -count=1 ./internal/handler/ -run "Test_P2"
```

## 项目结构

```
chain2plus1/
├── cmd/main.go                 # 程序入口（含优雅关闭）
├── internal/
│   ├── config/                 # 配置加载
│   ├── engine/                 # 链动分润引擎
│   ├── event/                  # WebSocket 事件/枢纽
│   ├── handler/                # HTTP 处理器 + 集成测试
│   ├── middleware/             # JWT鉴权/安全头/速率限制
│   ├── router/                 # 路由定义
│   ├── service/                # 业务逻辑 (支付/灵活用工)
│   └── testdb/                 # 测试数据库工具
├── pkg/
│   ├── database/               # MySQL 连接与迁移
│   ├── dto/                    # 数据传输对象
│   ├── jwt/                    # JWT 令牌
│   ├── logger/                 # Zap 结构化日志
│   ├── model/                  # GORM 数据模型
│   └── seed/                   # 种子数据
├── frontend/                   # React + Vite 前端
│   ├── src/
│   │   ├── lib/                # API 客户端 / Auth / WebSocket
│   │   ├── components/         # 布局 / Toast
│   │   └── pages/              # 8个页面组件
│   └── package.json
└── docs/                       # Swagger 文档
```

## API 文档

启动后端后访问: http://localhost:8080/swagger/index.html

### 默认管理员账号
- 用户名: `admin`
- 密码: `Admin@2024`

## 测试统计

```
P0: 18 个测试 (用户/订单/提现/分润)      ✅
P1:  8 个测试 (支付/灵活用工/评分/工时)   ✅
P2:  8 个测试 (链引擎/并发/边界/退款)     ✅
───────────────────────────────────────
总: 34 个测试, 0 失败
```
