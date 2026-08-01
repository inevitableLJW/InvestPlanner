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

根目录 `Makefile` 也提供 `make db-up`、`make server`、`make web`、`make test` 和 `make build`。

## 环境变量

后端变量见 `server/.env.example`：

- `MYSQL_DSN`：MySQL DSN；生产环境必须使用独立强密码并限制数据库权限。
- `WEB_ORIGIN`：允许携带 Cookie 的前端来源，必须是精确来源，不能使用通配符。
- `APP_ENV=production`：启用生产安全行为。
- `COOKIE_SECURE=true`：生产环境必须开启，并只通过 HTTPS 提供服务。
- `SESSION_TTL`：Go duration 格式的服务端会话有效期，例如 `168h`。

前端变量见 `web/.env.example`，生产部署可将 `VITE_API_BASE` 指向同源 `/api/v1`。

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
