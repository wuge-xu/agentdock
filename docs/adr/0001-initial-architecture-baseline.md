# ADR-0001：AgentDock 初始架构基线

- 状态：已接受
- 日期：2026-07-29
- 范围：系统总体架构

## 背景

AgentDock 需要接收多租户 Agent Task，将任务调度到隔离运行环境中执行，并统一管理状态、重试、取消、权限、日志、产物和审计。

系统同时涉及 HTTP API、数据库、可靠队列、Docker、Kubernetes、Python Agent Worker 和 MCP 工具调用。

如果没有明确的组件边界，任务状态、调度逻辑、执行逻辑和基础设施代码会快速耦合，导致系统难以测试、恢复和扩展。

## 决策一：采用单仓库

Go Control Plane、Python Agent Worker、数据库迁移、部署文件、测试和文档统一保存在 AgentDock 仓库中。

原因：

- 项目由一个团队统一维护
- Go 与 Python 组件需要共享接口契约
- 本地开发和端到端测试更加方便
- 现阶段拆分多个仓库没有明显收益

## 决策二：Go 负责控制平面

Go 组件负责：

- HTTP API
- 任务状态机
- 任务调度
- Redis 可靠队列协调
- Docker 和 Kubernetes 运行时管理
- 多租户与 RBAC
- MCP 注册与权限策略
- 审计日志
- Prometheus 指标

Go Control Plane 不在自身进程中执行不可信 Agent 代码。

## 决策三：Python 负责沙箱内 Agent 执行

Python Agent Worker 负责：

- 读取任务输入
- 执行 Agent 循环
- 请求 MCP 工具调用
- 输出结构化日志
- 生成结果和产物
- 返回标准退出状态

Python Worker 不负责全局任务调度，也不能直接裁决任务最终状态。

## 决策四：PostgreSQL 是事实来源

PostgreSQL 保存：

- 租户和用户
- 角色和权限
- Task
- TaskAttempt
- TaskEvent
- Artifact 元数据
- MCP Server
- ToolPolicy
- AuditLog

任务最终状态以 PostgreSQL 中的持久化记录为准。

Redis 不能作为任务事实状态的唯一来源。

## 决策五：Redis 负责临时调度状态

Redis 用于：

- pending 队列
- processing 队列
- retry 调度
- 执行租约
- Worker 心跳
- 并发计数
- 临时取消信号
- 短期幂等辅助

Redis 数据丢失后，系统应能够根据 PostgreSQL 恢复未完成任务。

## 决策六：Task 与 Attempt 分离

Task 表示用户提交的逻辑任务。

TaskAttempt 表示该任务的一次实际执行。

一个 Task 可以拥有多个 Attempt，用于记录：

- 重试次数
- 实际运行时
- 执行节点
- 开始和结束时间
- 失败原因
- 退出码
- 资源消耗
- 日志和产物

该设计可以保留完整执行历史，而不是在每次重试时覆盖原数据。

## 决策七：运行时使用统一接口

控制平面通过 Runtime Backend 抽象管理执行环境。

Runtime Backend 未来至少包含：

- Docker Runtime
- Kubernetes Job Runtime

上层任务调度逻辑不直接依赖某一种容器运行时。

## 决策八：先实现 Docker，再实现 Kubernetes

Docker 后端首先验证：

- 容器生命周期
- CPU 和内存限制
- PID 限制
- 执行超时
- 任务取消
- 日志采集
- 结果与产物回传
- 文件系统隔离
- 网络访问限制

核心执行链路稳定后，再实现 Kubernetes Job Backend。

## 决策九：默认最小权限

Agent 沙箱默认采用：

- 非 root 用户
- 只读根文件系统
- CPU 和内存限制
- PID 数量限制
- 执行时间限制
- 禁止未授权工具
- 禁止未授权网络访问
- 禁止提权
- 关键操作写入审计日志

任何额外权限都必须由明确的任务策略授予。

## 正面影响

- 核心状态机可以独立测试
- 任务事实不会依赖 Redis
- 每次重试都能保留执行记录
- 可以增加新的 Runtime Backend
- Go Control Plane 不直接执行不可信代码
- 安全策略可以由平台统一治理

## 代价和风险

- PostgreSQL 与 Redis 之间需要处理一致性
- Task 与 Attempt 分离增加了数据模型复杂度
- 取消和正常完成之间存在并发竞争
- Runtime Backend 需要统一不同运行时的状态语义
- Docker Socket 本身具有高权限，需要额外设计安全边界
- Kubernetes Job 的状态需要映射为 AgentDock 状态

## 后续需要单独决策

- Go HTTP 路由方案
- PostgreSQL 驱动
- 数据库迁移工具
- ID 生成策略
- 状态机并发控制
- PostgreSQL 与 Redis 一致性方案
- Docker Runtime 接口
- Kubernetes Job 状态映射
- MCP 授权模型
- 网络出口控制方案
- Artifact 存储方案
