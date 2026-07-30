# 阶段 1：最小 Control Plane 需求契约

- 状态：已冻结
- 日期：2026-07-30
- 范围：最小任务持久化控制平面

## 目标

阶段 1 建立第一个可以运行、测试和展示的 AgentDock Control Plane。

完整链路：

客户端提交 Task
→ Go HTTP API 校验请求
→ PostgreSQL 持久化 Task
→ 返回任务标识
→ 客户端根据 Task ID 查询

本阶段不执行 Agent，也不使用 Redis 调度任务。

## 包含范围

- Go Control Plane 启动入口
- 配置读取与校验
- JSON 结构化日志
- 请求 ID 传播
- 存活检查
- 就绪检查
- PostgreSQL 连接
- 数据库迁移
- Task 创建
- Task 查询
- 租户级数据隔离
- 幂等创建
- 单元测试
- PostgreSQL 集成测试
- Docker Compose 本地基础设施

## 不包含范围

- Redis 可靠队列
- Dispatcher
- Executor
- Python Agent Worker
- Docker Sandbox
- Kubernetes Job
- 任务取消
- 自动重试
- Dead Letter Queue
- JWT 与 RBAC
- MCP Server
- 网络访问控制
- Artifact 存储

## HTTP 接口

### GET /health/live

用途：

确认 Control Plane 进程仍在运行。

该接口不检查 PostgreSQL 或其他外部依赖。

成功响应状态码：

200

成功响应：

{
  "status": "alive"
}

### GET /health/ready

用途：

确认当前实例能够正常提供任务 API。

检查内容：

- 配置已经通过校验
- PostgreSQL 连接正常

成功响应状态码：

200

成功响应：

{
  "status": "ready",
  "checks": {
    "postgres": "up"
  }
}

依赖异常响应状态码：

503

### POST /api/v1/tasks

请求头：

- X-Tenant-ID：必填
- Idempotency-Key：必填
- X-Request-ID：选填

请求体：

{
  "name": "repository-analysis",
  "input": {
    "repository": "example/repository"
  },
  "max_attempts": 3
}

校验规则：

- X-Tenant-ID 必须是合法 UUID
- Idempotency-Key 长度为 1 到 128
- name 长度为 1 到 128
- input 必须是 JSON 对象
- max_attempts 范围为 1 到 10

首次创建响应状态码：

201

幂等重复响应状态码：

200

响应字段：

- id
- tenant_id
- name
- input
- status
- max_attempts
- version
- created_at
- updated_at

### GET /api/v1/tasks/{task_id}

请求头：

- X-Tenant-ID：必填
- X-Request-ID：选填

规则：

- task_id 必须是合法 UUID
- 只能返回属于当前 tenant_id 的任务
- 不向调用方泄露其他租户的任务是否存在

成功响应状态码：

200

不存在或不属于当前租户：

404

## Task 初始状态

阶段 1 只创建 created 状态的 Task。

允许的状态事实：

created：任务已经持久化，但尚未进入 Redis 队列。

阶段 2 实现 Dispatcher 后再增加：

created → queued

## Task 数据字段

| 字段 | 类型 | 说明 |
|---|---|---|
| id | UUID | Task 主键 |
| tenant_id | UUID | 所属租户 |
| idempotency_key | VARCHAR(128) | 租户范围内的幂等键 |
| name | VARCHAR(128) | 任务名称 |
| input | JSONB | 任务输入 |
| status | VARCHAR | 当前状态 |
| max_attempts | SMALLINT | 最大执行次数 |
| version | BIGINT | 乐观锁版本 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

## 数据库约束

- PRIMARY KEY (id)
- UNIQUE (tenant_id, idempotency_key)
- max_attempts BETWEEN 1 AND 10
- version 大于等于 1
- status 初始值为 created
- tenant_id 不允许为空
- input 不允许为空

## 多租户边界

阶段 1 使用 X-Tenant-ID 传递租户身份。

这只是开发阶段的身份传播方式，不构成安全认证。

Repository 查询必须同时使用：

- task_id
- tenant_id

不能先按 task_id 查询，再在应用层判断 tenant_id。

后续认证与 RBAC 阶段会从受信任凭证解析 tenant_id。

## 幂等语义

同一个租户使用相同 Idempotency-Key 重复提交时：

- 不创建新 Task
- 返回原有 Task
- 不修改原任务输入
- 不增加版本号

不同租户可以使用相同的 Idempotency-Key。

## 错误响应

所有错误采用统一结构：

{
  "error": {
    "code": "invalid_request",
    "message": "request validation failed",
    "request_id": "..."
  }
}

阶段 1 计划错误码：

- invalid_request
- invalid_tenant
- invalid_task_id
- task_not_found
- database_unavailable
- internal_error

## 可观测性要求

每个请求记录：

- request_id
- method
- path
- status_code
- duration_ms

日志不得直接记录完整 input，避免后续任务输入中的敏感信息泄露。

## 验收标准

- Control Plane 可以独立启动
- PostgreSQL可以通过Docker Compose启动
- /health/live 返回200
- PostgreSQL正常时 /health/ready 返回200
- PostgreSQL停止时 /health/ready 返回503
- 可以创建Task
- 可以根据Task ID和Tenant ID查询Task
- 其他Tenant查询返回404
- 相同幂等键不会创建重复Task
- 非法输入返回稳定的JSON错误
- 单元测试通过
- PostgreSQL集成测试通过
- go test -race ./... 通过
