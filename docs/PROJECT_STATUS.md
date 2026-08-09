# AgentDock 项目状态
* 最后更新：2026-08-09
* 当前阶段：阶段 1——最小任务 Control Plane
* 当前步骤：1-F——可运行 Go Control Plane 服务骨架（已完成）
* 下一步骤：1-G——PostgreSQL 基础设施、连接池与 Readiness
## 项目定位
AgentDock 是一个面向多租户 Agent 工作负载的运行时与安全执行平台。
核心链路：
用户提交 Agent Task
→ Go Control Plane
→ PostgreSQL 持久化任务状态
→ Redis 可靠队列与调度
→ Docker / Kubernetes 沙箱
→ Python Agent Worker
→ MCP / Tool 调用
→ 日志、产物、结果与审计
项目重点不是普通 Agent CRUD，而是：
* 可靠任务执行
* 状态一致性
* 多租户隔离
* 容器化 Runtime
* 资源限制
* 网络与工具治理
* 故障恢复
* Kubernetes 执行后端
* 可观测性与审计
## 当前架构路线
第一阶段：
Client
→ Go HTTP API
→ PostgreSQL
→ Task 创建与查询
第二阶段：
PostgreSQL
→ Dispatcher
→ Redis pending
→ processing
→ Worker
→ ACK / Retry / Dead Letter / Lease Recovery
第三阶段：
Control Plane
→ Docker Runtime
→ Python Agent Worker
→ 日志 / 结果 / Artifact
第四阶段：
Runtime Policy
→ CPU / Memory / Timeout
→ Network Policy
→ Tool Allowlist
→ Tenant Policy
第五阶段：
DockerRuntime
→ Runtime Backend 抽象
→ KubernetesJobRuntime
→ Kubernetes Job / NetworkPolicy / Observability
\---
# 已完成阶段
## 阶段 0——工程基线与架构契约
已完成：
* 明确 AgentDock 项目名称、定位与核心链路
* 完成本地系统资源检查
* 打通 Docker Desktop 与 WSL 集成
* 验证 Docker Engine 和 Docker Compose
* 验证 K3s 单节点集群
* 建立 Python 3.12 独立 Worker 虚拟环境
* 安装 GCC 与 Go Race Detector 所需依赖
* 统一 kubectl 使用方式
* 初始化 Git 仓库
* 初始化 Go Module
* 建立项目基础目录
* 创建 README
* 创建 ADR-0001 初始架构决策记录
* 建立 GitHub 公开仓库
* 建立 main 与 origin/main 跟踪关系
* 将工程基线推送 GitHub
阶段 0 验收环境：
* Ubuntu 24.04.1 on WSL2
* Go 1.25.0
* Python Worker 3.12.3
* Docker Engine 29.5.2
* Docker Compose 5.1.3
* Kubernetes K3s 1.35.5
* Git 默认分支 main
* GitHub 仓库：wuge-xu/agentdock
* 仓库可见性：Public
主要提交：
* `e2e6d40`：初始化 AgentDock 工程基线
* `9e0a4ac`：关闭阶段 0 工程基线
\---
# 阶段 1——最小任务 Control Plane
阶段目标：
Client 提交 Task
→ Go HTTP API 校验
→ PostgreSQL 持久化
→ 返回 Task ID
→ 根据 Task ID 查询 Task
当前阶段尚未全部结束。
已经完成 Control Plane 服务基础、配置、日志和 HTTP Runtime，下一步开始 PostgreSQL 数据闭环。
## 1-A——需求与接口契约
已完成：
* 冻结阶段 1 最小需求范围
* Task 初始状态确定为 `created`
* `created → queued` 延迟到 Redis Dispatcher 阶段
* 定义租户级幂等键语义
* Task 查询必须同时携带 `task_id` 与 `tenant_id`
* 跨租户查询返回 404，避免资源存在性泄漏
* 区分 Liveness 与 Readiness 职责
* PostgreSQL 作为任务持久化事实来源
* Redis 不进入阶段 1 核心链路
计划 HTTP API：
### GET /health/live
只验证 Control Plane 进程是否存活，不访问数据库。
### GET /health/ready
验证：
* 配置有效
* PostgreSQL 可连接
结果：
* Ready：HTTP 200
* Dependency Down：HTTP 503
### POST /api/v1/tasks
Header：
* `X-Tenant-ID`
* `Idempotency-Key`
* `X-Request-ID` 可选
Body：
* `name`
* `input`
* `max_attempts`
语义：
* 首次创建返回 201
* 同租户相同 Idempotency-Key 返回已有 Task，HTTP 200
### GET /api/v1/tasks/{task_id}
必须使用：
* task ID
* tenant ID
联合查询。
跨租户访问统一返回 404。
\---
## 1-B——技术选型
已完成 ADR-0002。
当前确定：
| 模块         | 技术                             |
| ---------- | ------------------------------ |
| HTTP       | Go 标准库 `net/http` + `ServeMux` |
| 日志         | `log/slog` + `JSONHandler`     |
| 配置         | 自定义强类型环境变量解析                   |
| PostgreSQL | `pgx/v5` + `pgxpool`           |
| SQL        | 第一阶段手写参数化 SQL                  |
| Migration  | Goose v3 SQL Migration         |
| 测试         | Go testing + Race Detector     |
| Runtime    | Docker 优先，Kubernetes Job 后续    |
数据库 Migration 不与应用启动自动绑定。
\---
## 1-C / 1-D——强类型配置系统
已完成：
* `internal/config`
* 强类型环境变量解析
* HTTP 配置
* 日志配置
* PostgreSQL 连接池配置
* Shutdown Timeout
* 环境变量缺失与显式空值区分
* URL 格式校验
* Duration 范围校验
* Connection Pool 范围校验
* 跨字段校验
* 单元测试
* 非法输入测试
主要配置项：
* `AGENTDOCK_HTTP_ADDRESS`
* `AGENTDOCK_HTTP_READ_HEADER_TIMEOUT`
* `AGENTDOCK_HTTP_READ_TIMEOUT`
* `AGENTDOCK_HTTP_WRITE_TIMEOUT`
* `AGENTDOCK_HTTP_IDLE_TIMEOUT`
* `AGENTDOCK_LOG_LEVEL`
* `AGENTDOCK_DATABASE_URL`
* `AGENTDOCK_DATABASE_MAX_CONNS`
* `AGENTDOCK_DATABASE_MIN_CONNS`
* `AGENTDOCK_DATABASE_CONNECT_TIMEOUT`
* `AGENTDOCK_SHUTDOWN_TIMEOUT`
统一质量入口：
* `make check`
* `make test`
* `make test-race`
* `make vet`
主要提交：
* `13b283f`：实现强类型配置系统
\---
## 1-E——JSON 结构化日志
已完成：
* 基于 `log/slog` 创建统一日志入口
* 使用 JSONHandler
* 默认输出 stdout
* 自动附加 `service`
* 支持附加 `version`
* 支持 Component 派生日志器
* 日志级别过滤
* 日志单元测试
* Race Detector 验证
默认服务名：
`agentdock-control-plane`
主要提交：
* `a108113`：实现 JSON 结构化日志基础
\---
## 1-F——可运行 Go Control Plane 服务骨架
状态：已完成并真实运行验收。
### HTTP Response
已实现：
* JSON Response Writer
* 统一 JSON Error Envelope
错误格式：
`{"error":{"code":"...","message":"...","request_id":"..."}}`
当前已使用：
* `route_not_found`
* `method_not_allowed`
* `request_id_generation_failed`
### Liveness
已实现：
`GET /health/live`
返回：
`{"status":"alive"}`
Liveness 不访问 PostgreSQL。
### Request ID
已实现：
* Header：`X-Request-ID`
* 客户端提供时透传
* 未提供时自动生成
* 使用 `crypto/rand`
* 生成 16 Byte 随机值
* Hex 编码为 32 字符 Request ID
* Request ID 写入 `context.Context`
* Response Header 返回相同 Request ID
* JSON Error 中携带 Request ID
### HTTP Access Log
已实现统一访问日志 Middleware。
记录字段：
* `request_id`
* `method`
* `path`
* `status_code`
* `duration_ms`
访问日志继承：
* `service`
* `version`
* `component=http`
Query String 默认不进入 Access Log。
Middleware 顺序：
RequestIDMiddleware
→ AccessLogMiddleware
→ ServeMux
从而确保 Access Log 能读取已经进入 Context 的 Request ID。
### HTTP Router
已实现：
* Go 1.22+ Method + Path ServeMux Pattern
* `GET /health/live`
* `/health/live` 非 GET 请求统一 JSON 405
* 未知路由统一 JSON 404
### HTTP Server
已显式创建 `http.Server`。
配置：
* Address
* ReadHeaderTimeout
* ReadTimeout
* WriteTimeout
* IdleTimeout
避免直接使用无完整超时控制的简单 `http.ListenAndServe` 启动方式。
### Control Plane 启动入口
已实现：
`cmd/control-plane/main.go`
启动链路：
Config.Load
→ JSON Logger
→ HTTP Router
→ HTTP Server
→ ListenAndServe
服务生命周期支持：
* SIGINT
* SIGTERM
* Graceful Shutdown
* Shutdown Timeout
* Server 启动错误传播
* 非正常 HTTP Server 退出检测
### 真实进程运行验收
2026-08-09 使用：
`127.0.0.1:18081`
真实启动 AgentDock Control Plane。
验收结果：
#### Liveness
`GET /health/live`
结果：
* HTTP 200
* `Content-Type: application/json`
* Request ID 正确透传
* Body 返回 `{"status":"alive"}`
#### JSON 404
访问：
`GET /does-not-exist`
结果：
* HTTP 404
* Error Code：`route_not_found`
* Request ID 正确进入 Error Envelope
#### JSON 405
访问：
`POST /health/live`
结果：
* HTTP 405
* Error Code：`method_not_allowed`
* Request ID 正确进入 Error Envelope
#### Access Log
真实日志已验证包含：
* service
* version
* component
* request_id
* method
* path
* status_code
* duration_ms
#### Graceful Shutdown
向真实 Control Plane 发送：
`SIGTERM`
结果：
* 记录 `shutdown signal received`
* HTTP Server 正常 Shutdown
* 记录 `control plane stopped`
* 进程退出码为 0
阶段 1-F 已完成真实进程级运行闭环。
\---
# 当前测试状态
当前模块：
* `cmd/control-plane`
* `internal/config`
* `internal/platform/logging`
* `internal/transport/http`
当前 HTTP 测试覆盖：
* Access Log 显式状态码
* Access Log 隐式 HTTP 200
* Access Log Duration 防负值
* Request ID 客户端透传
* Request ID 自动生成
* Request ID Trim
* Request ID 生成失败
* Request ID 随机性
* Liveness
* Method Not Allowed
* JSON Not Found
* HTTP Server 配置映射
已通过：
* `go test`
* `go vet`
* `make check`
* `make test-race`
* `git diff --check`
* Control Plane Build
* 真实进程运行验收
\---
# 已接受架构决定
| 编号     | 架构决定                                     |
| ------ | ---------------------------------------- |
| AD-001 | 单仓库管理 Go、Python、部署、测试与文档                 |
| AD-002 | Go 负责 Control Plane                      |
| AD-003 | Python 负责沙箱内 Agent 执行                    |
| AD-004 | PostgreSQL 是任务持久化事实来源                    |
| AD-005 | Redis 负责临时可靠队列与调度状态                      |
| AD-006 | Task 与 TaskAttempt 分离                    |
| AD-007 | 使用 Runtime Backend 抽象                    |
| AD-008 | 先实现 Docker Runtime，再实现 Kubernetes Job    |
| AD-009 | Agent 执行环境默认最小权限                         |
| AD-010 | 阶段 1 HTTP 使用 Go 标准库 net/http             |
| AD-011 | 统一结构化日志使用 log/slog                       |
| AD-012 | 配置系统使用自定义强类型环境变量解析                       |
| AD-013 | PostgreSQL 使用 pgx/v5 + pgxpool           |
| AD-014 | 第一阶段使用手写参数化 SQL                          |
| AD-015 | Migration 使用 Goose v3                    |
| AD-016 | Migration 与应用启动生命周期分离                    |
| AD-017 | Liveness 不依赖外部服务                         |
| AD-018 | Readiness 负责 PostgreSQL Dependency Check |
\---
# 已冻结 Task 第一版数据契约
Task 计划字段：
* `id UUID`
* `tenant_id UUID`
* `idempotency_key VARCHAR(128)`
* `name VARCHAR(128)`
* `input JSONB`
* `status`
* `max_attempts SMALLINT`
* `version BIGINT`
* `created_at TIMESTAMPTZ`
* `updated_at TIMESTAMPTZ`
唯一约束：
`(tenant_id, idempotency_key)`
初始状态：
`created`
Redis Dispatcher 完成后：
`created → queued`
\---
# 当前未完成事项
## 阶段 1 后半部分
1\. 创建 PostgreSQL Docker Compose 基础设施
2\. 接入 `pgx/v5`
3\. 建立 `pgxpool`
4\. 实现连接超时
5\. 实现 PostgreSQL Ping
6\. 实现 `/health/ready`
7\. 创建 Goose Migration
8\. 创建 Task 表
9\. 定义 Task Domain Model
10\. 实现 Task Repository
11\. 实现 Task Create
12\. 实现 Task Query
13\. 实现租户隔离
14\. 实现 Idempotency-Key
15\. 增加 PostgreSQL Integration Test
16\. 完成阶段 1 E2E 验收
## 阶段 2
1\. Redis 基础设施
2\. pending queue
3\. processing queue
4\. ACK
5\. retry
6\. dead-letter queue
7\. lease
8\. Worker Crash Recovery
9\. Redis 丢失后的 PostgreSQL 重建机制
## 阶段 3
1\. Docker Runtime
2\. Python Agent Worker
3\. Worker Protocol
4\. CPU Limit
5\. Memory Limit
6\. Timeout
7\. Cancel
8\. Streaming Log
9\. Artifact
10\. Cold Start Benchmark
\---
# 下一里程碑
当前下一条真实链路：
PostgreSQL Container
→ pgxpool
→ `/health/ready`
→ Goose Migration
→ Task Table
→ POST `/api/v1/tasks`
→ PostgreSQL Persist
→ GET `/api/v1/tasks/{task_id}`
→ Tenant Isolation
→ Idempotency
完成后，阶段 1 最小 Control Plane 才正式关闭。
\---
# 风险与待设计事项
* PostgreSQL 与 Redis 状态一致性策略
* Dispatcher 如何保证 Task 不重复入队
* Redis 丢失后的任务重建
* Worker ACK 与结果持久化顺序
* Worker Crash 后 Lease Recovery
* Task Cancel 与正常完成之间的竞态
* Docker Socket 权限边界
* Docker Container 是否使用 Rootless / Remote Runtime
* Agent 网络访问默认策略
* Tool Allowlist 与 MCP 权限模型
* Artifact 生命周期
* Kubernetes Job 与 TaskAttempt 状态映射
* Kubernetes Job Watch 断连恢复
* 多租户资源配额
* Audit Log 不可抵赖性
\---
# 当前阶段结论
AgentDock 已从“工程目录与架构设计”推进为一个真正可运行的 Go Control Plane。
当前已经具备：
Config
→ Structured Logging
→ Request ID
→ HTTP Access Log
→ Router
→ Liveness
→ Explicit HTTP Timeout
→ Process Lifecycle
→ Graceful Shutdown
下一阶段不再继续增加通用基础组件。
直接进入：
PostgreSQL
→ Task Persistence
→ Tenant Isolation
→ Idempotency
使 AgentDock 从“可运行服务”继续推进为“真正能够管理 Agent Task 的 Control Plane”。
