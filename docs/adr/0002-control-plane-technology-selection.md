# ADR-0002：阶段 1 Control Plane 技术选型

- 状态：已接受
- 日期：2026-07-30
- 范围：HTTP、配置、日志、PostgreSQL与数据库迁移

## 背景

阶段 1 需要实现最小 Go Control Plane，包括健康检查、Task 创建、Task 查询、PostgreSQL持久化、租户隔离和幂等创建。

本阶段需要选择基础技术组件，但必须避免过早引入复杂框架和代码生成系统。

选型原则：

- 优先使用 Go 标准库
- 保持依赖数量较少
- 核心链路必须可以直接理解
- 支持单元测试和集成测试
- 支持未来 Docker 和 Kubernetes 部署
- 不阻碍后续增加 Redis、认证和可观测性

## 决策一：HTTP 使用 net/http ServeMux

阶段 1 使用 Go 标准库 net/http。

路由使用方法和路径模式：

- GET /health/live
- GET /health/ready
- POST /api/v1/tasks
- GET /api/v1/tasks/{task_id}

路径参数通过 request.PathValue 获取。

暂不采用：

- Gin
- Echo
- Fiber
- Chi

原因：

- 当前接口数量较少
- Go 1.25 的 ServeMux 已支持方法和路径通配符
- 可以减少第三方依赖
- 有利于理解 Handler 和 Middleware 模型
- 可以直接使用 httptest 测试

如果后续路由规模、子路由或中间件管理明显复杂化，再重新评估是否引入路由库。

## 决策二：HTTP Server 显式配置超时

Control Plane 不使用裸 http.ListenAndServe。

需要创建 http.Server，并显式配置：

- ReadHeaderTimeout
- ReadTimeout
- WriteTimeout
- IdleTimeout

进程收到终止信号时，调用 Shutdown 完成优雅关闭。

## 决策三：日志使用 log/slog

日志使用 Go 标准库 log/slog。

默认使用 JSONHandler 输出到标准输出。

每条请求日志至少包含：

- request_id
- method
- path
- status_code
- duration_ms

服务日志根据场景增加：

- component
- operation
- task_id
- tenant_id
- error

日志不能记录：

- 数据库密码
- API Token
- 完整 Task input
- MCP Credential

## 决策四：配置使用自定义强类型结构

不引入 Viper 或其他配置框架。

配置从环境变量读取，并经过三个步骤：

1. 读取
2. 类型转换
3. 业务校验

计划中的第一版配置：

- AGENTDOCK_HTTP_ADDRESS
- AGENTDOCK_LOG_LEVEL
- AGENTDOCK_DATABASE_URL
- AGENTDOCK_DATABASE_MAX_CONNS
- AGENTDOCK_DATABASE_MIN_CONNS
- AGENTDOCK_DATABASE_CONNECT_TIMEOUT
- AGENTDOCK_SHUTDOWN_TIMEOUT

Config对象创建成功后不再修改。

必须区分：

- 环境变量不存在
- 环境变量存在但为空
- 环境变量格式错误
- 环境变量超出允许范围

## 决策五：PostgreSQL使用pgx/v5

使用 github.com/jackc/pgx/v5。

Control Plane使用pgxpool连接池，而不是共享单个pgx.Conn。

原因：

- HTTP服务存在并发数据库访问
- pgx原生支持PostgreSQL类型
- 能直接处理JSONB和UUID
- 支持连接池配置和健康检查
- 不需要经过ORM抽象

## 决策六：阶段 1 手写 SQL

阶段 1 不引入：

- GORM
- Ent
- sqlc
- SQL Query Builder

Repository实现直接编写参数化SQL。

原因：

- 当前查询数量少
- 需要清楚理解租户查询条件
- 需要直接处理唯一约束和幂等冲突
- 需要理解事务与错误码
- 避免代码生成增加初期复杂度

当查询规模显著扩大时，可以单独评估sqlc。

## 决策七：数据库迁移使用Goose v3

迁移文件使用SQL格式，存放于：

migrations/

文件采用顺序版本和描述性名称，例如：

00001_create_tasks.sql

每个迁移包含：

- goose Up
- goose Down

迁移原则：

- Schema变更进入Git
- 不手工修改共享数据库结构
- 每次迁移只承担一个明确目的
- 迁移必须先在本地和集成测试数据库验证
- 已经推送并执行的迁移不允许直接重写

## 决策八：应用启动时不自动迁移

Control Plane启动过程不自动执行数据库迁移。

迁移由显式命令完成。

本地开发：

make migrate-up

持续集成：

启动测试数据库后显式运行迁移

Kubernetes：

使用独立Migration Job

原因：

- 避免多个应用副本同时迁移
- 将Schema变更与应用生命周期分离
- 迁移失败时不会表现为普通应用启动失败
- 更容易审计数据库变更

## 决策九：阶段 1 不使用Redis

Redis尚未进入阶段 1 的运行链路。

Task创建完成后的真实状态为created。

Redis、Dispatcher和created到queued的状态转换在阶段 2 实现。

## 影响

### 正面影响

- 第三方依赖较少
- HTTP核心链路清晰
- 配置错误可以在启动阶段快速失败
- 日志天然适配容器环境
- PostgreSQL能力不会被ORM隐藏
- 数据库迁移可独立执行和审计
- 后续可以平滑加入Redis与Runtime Backend

### 代价

- 需要自己实现HTTP中间件
- 需要自己实现配置解析和校验
- Repository需要手写扫描逻辑
- 没有ORM自动生成表结构
- 没有框架自动完成请求绑定和错误响应

这些代价在阶段 1 的范围内可控，并有助于理解核心实现。

## 后续需要验证

- ServeMux路由冲突和404行为
- HTTP Server超时配置
- Request ID生成与传播
- pgxpool启动连接验证
- PostgreSQL不可用时的就绪检查
- PostgreSQL唯一约束冲突处理
- Goose Up和Down迁移
- 应用关闭时连接池释放
