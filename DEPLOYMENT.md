# 部署文档

## 环境要求

| 组件 | 版本 | 说明 |
|------|------|------|
| Go | 1.26+ | 编译后端 |
| MySQL | 5.7+ | 主数据库 |
| Node.js | 20+ | 构建前端 (可选) |
| Docker | 24+ | 容器化部署 (推荐) |
| Docker Compose | 2.20+ | 编排服务 |

## 1. Docker 部署（推荐）

```bash
# 克隆代码
git clone https://github.com/yang-lei-cmd/chain2plus1.git
cd chain2plus1

# 一键启动
docker compose up -d

# 查看日志
docker compose logs -f

# 停止
docker compose down
```

启动后访问:
- API: http://localhost:8080
- Swagger: http://localhost:8080/swagger/index.html
- 前端: http://localhost:5173 (需额外启动)

### 环境变量

通过 `docker-compose.yml` 或 `.env` 文件配置:

```env
PORT=8080                    # API端口
MODE=release                 # release/debug
DB_HOST=mysql                # 数据库地址
DB_PORT=3306                 # 数据库端口
DB_USER=root                 # 数据库用户
DB_PASSWORD=Linqi@2024       # 数据库密码
DB_NAME=chain2plus1          # 数据库名
JWT_SECRET=Linqi@2024        # JWT密钥
JWT_EXPIRE_DAYS=7            # JWT过期天数
LOG_LEVEL=info               # 日志级别
```

## 2. 手动部署

### 后端

```bash
# 1. 创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS chain2plus1 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"

# 2. 设置环境变量 (Windows PowerShell)
$env:DB_HOST="127.0.0.1"
$env:DB_PASSWORD="Linqi@2024"
$env:JWT_SECRET="Linqi@2024"

# 3. 编译并启动
go build -o server.exe ./cmd/main.go
./server.exe
```

### 前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build

# 预览构建产物
npm run preview
```

构建产物在 `frontend/dist/`，可用 Nginx 托管:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    root /path/to/chain2plus1/frontend/dist;
    index index.html;

    # API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket 代理
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # SPA 路由
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

## 3. 数据库备份

```bash
# 备份
docker exec chain2plus1-mysql mysqldump -u root -pLinqi@2024 chain2plus1 > backup_$(date +%Y%m%d).sql

# 恢复
docker exec -i chain2plus1-mysql mysql -u root -pLinqi@2024 chain2plus1 < backup_20260703.sql
```

## 4. 健康检查

```bash
# 存活检查
curl http://localhost:8080/health
# → {"status":"ok","message":"Chain2Plus1 API v2 running"}

# 就绪检查 (含数据库)
curl http://localhost:8080/ready
# → {"status":"ok","database":"connected"}
```

## 5. 监控建议

- **日志**: Docker 日志 + 文件采集 (Filebeat → Elasticsearch)
- **指标**: Prometheus + Grafana (Go runtime metrics)
- **告警**: 健康检查失败 → 钉钉/飞书/邮件通知
- **备份**: 每日自动备份 MySQL，保留7天

## 6. 扩容建议

- 数据库: MySQL 主从复制，读写分离
- API: 多实例部署，Nginx 负载均衡
- 缓存: Redis 缓存热数据（用户 session、商品列表）
- 队列: RabbitMQ/Redis Stream 异步处理分润计算
