# Invest Planner 定投计划

一个以“先创建计划，再按计划记录月度收支”为核心的 Web 应用。支持账户登录、多定投计划、可配置投资标的、按付款 App 汇总支出、服务端权威计算、实际投入追踪、历史和计划级统计。

## 技术栈

- 前端：React 19、TypeScript、Vite、React Router
- 后端：Go、Gin、GORM
- 数据库：MySQL 8.4
- 金额全部以整数分存储，比例以基点（10,000 = 100%）存储

## 本地启动

要求 Docker Desktop、Go 1.24+ 和 Node.js 20+。

```bash
docker compose up -d mysql
cp server/.env.example server/.env
cp web/.env.example web/.env

cd server
go mod download
set -a; . ./.env; set +a; go run ./cmd/api
```

另开终端：

```bash
cd web
npm install
npm run dev
```

浏览器访问 `http://localhost:5173`。Vite 开发服务器将 `/api` 代理到 `http://localhost:8080`。

根目录 `Makefile` 也提供 `make db-up`、`make dev-api`、`make dev-web`、`make test` 和 `make build`。

## Docker 一键部署

整套应用由三个容器组成：`web`（唯一对外入口）、`api` 和 `mysql`。数据库端口不会暴露到宿主机，前端通过同源 `/api` 访问后端。

```bash
cp .env.docker.example .env
# 编辑 .env：至少修改 APP_URL、MYSQL_DATA_DIR 和 MYSQL_PASSWORD
docker compose up -d --build
docker compose ps
```

访问 `.env` 中的 `APP_URL`（默认端口 `3000`）。升级时执行：

```bash
git pull
docker compose up -d --build
```

应用数据仅存放在 `MYSQL_DATA_DIR`；删除或重建容器不会删除该目录。不要使用 `docker compose down -v`，升级或迁移前应先备份数据库。

## TrueNAS SCALE 25.10.1

推荐先在 TrueNAS 创建一个应用数据集，例如 `/mnt/tank/apps/invest-planner/mysql`，Dataset Preset 选择 **Apps**，不要把数据库直接放进 SMB 共享目录。若容器日志提示 `permission denied`，请先检查该数据集的 Apps ACL。然后选择以下任一部署方式：

### 方式一：通过 SSH 从源码构建

将仓库克隆到 NAS 数据集，复制并修改 `.env.docker.example`，然后执行上面的 `docker compose up -d --build`。这种方式不依赖容器仓库权限。

### 方式二：在 TrueNAS 界面安装

1. 推送本仓库后，确认 GitHub Actions 的 `Container images` 工作流成功，并将两个 GHCR Package 设置为 Public。
2. 打开 **Apps > Discover Apps > 右上角菜单 > Install via YAML**，应用名填写 `invest-planner`。
3. 粘贴 [`deploy/truenas-compose.yml`](deploy/truenas-compose.yml) 的内容。
4. 全局替换模板中的三个值：`REPLACE_WITH_POOL`、`REPLACE_WITH_TRUENAS_IP`、`REPLACE_WITH_A_LONG_RANDOM_PASSWORD`。密码建议只使用至少 32 位的 ASCII 字母和数字，避免 Compose 将 `$` 等字符当作变量。
5. 保存，等待三个容器健康后访问 `http://TRUENAS_IP:3000`。

模板会拉取 `linux/amd64` 或 `linux/arm64` 的 GHCR 镜像，适配常见 TrueNAS 主机架构。端口 `3000` 如被占用，可同时修改 `web.ports` 左侧端口和 `WEB_ORIGIN` 中的端口。

如果通过 HTTPS 反向代理对外提供服务，请把 `WEB_ORIGIN`（或 `.env` 中的 `APP_URL`）改为精确的 `https://域名`，并设置 `COOKIE_SECURE=true`。不要在公网直接暴露 MySQL 或本项目的 HTTP 端口。

首次初始化后再修改 `MYSQL_PASSWORD` 不会自动修改数据库内已有账号密码；如需轮换密码，请先在 MySQL 中修改账号，再同步更新 Compose 配置。

健康检查：入口为 `/healthz`，API 容器为 `http://api:8080/healthz`。排障可查看：

```bash
docker compose ps
docker compose logs --tail=100 mysql api web
```

## 环境变量

后端变量见 `server/.env.example`：

- `MYSQL_DSN`：MySQL DSN；生产环境必须使用独立强密码并限制数据库权限。
- `WEB_ORIGIN`：允许携带 Cookie 的前端来源，必须是精确来源，不能使用通配符。
- `APP_ENV=production`：启用生产安全行为。
- `COOKIE_SECURE=true`：生产环境必须开启，并只通过 HTTPS 提供服务。
- `SESSION_TTL`：Go duration 格式的服务端会话有效期，例如 `168h`。

前端变量见 `web/.env.example`，生产部署默认使用同源 `/api/v1`；如需覆盖可设置 `VITE_API_BASE_URL`。

不要提交 `.env`、真实 DSN、密码或会话令牌。日志和错误响应不会输出密码、密码哈希、原始会话令牌或其他用户的计划数据。

## 数据库迁移

API 启动时执行幂等、版本化迁移，版本记录在 `schema_migrations`。部署顺序：

1. 备份数据库并验证恢复流程。
2. 部署并启动新后端，让向前兼容的迁移完成。
3. 检查 `/healthz`。
4. 部署前端静态资源。

当前首版迁移只创建新表。回滚应用版本时可保留这些兼容表；如必须删除，请先备份并在维护窗口手工执行，避免自动破坏用户历史记录。

## API 约定

所有业务接口位于 `/api/v1`。认证使用服务端保存、可撤销、会过期的随机会话，浏览器仅保存 HttpOnly、SameSite=Lax Cookie。

- `POST /auth/register`、`POST /auth/login`、`GET /auth/me`、`POST /auth/logout`
- `/plans` 管理计划；`/plans/:planID/months`、`/stats` 是计划嵌套资源
- `/expense-sources` 管理用户级付款平台
- `400` 表示输入校验失败，`401` 表示登录失效，`404` 同时用于不存在或不属于当前用户的资源，`409` 表示版本冲突，`5xx` 可重试

所有写请求都校验 Origin；开发环境只对配置的来源启用携带凭据的 CORS。计划和月度记录使用版本号做乐观并发控制。

## 计算规则

```text
可投入基数 = max(收入 - 平台支出合计 - 预留金额, 0)
建议投入 = floor(可投入基数 × 本月投入比例 / 取整单位) × 取整单位
```

投资标的数量不固定。启用标的至少一个且比例合计必须为 100%；使用最大余数法分配取整尾差。历史月份保存标的名称、顺序和比例快照，之后修改计划不会改写历史结构。

## 验证

```bash
cd web
npm test -- --run
npm run lint
npm run build

cd ../server
go test ./...
go vet ./...
```

真实 MySQL 集成测试：

```bash
MYSQL_TEST_DSN='invest_planner:invest_planner@tcp(127.0.0.1:3306)/invest_planner?charset=utf8mb4&parseTime=True&loc=Local' go test ./integration -v
```

集成测试会清空该 DSN 中的业务表，必须使用专用测试数据库，禁止指向生产库。
