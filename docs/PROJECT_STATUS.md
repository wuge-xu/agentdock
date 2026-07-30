# AgentDock 项目状态

- 最后更新：2026-07-30
- 当前阶段：阶段 1——最小任务 Control Plane
- 当前步骤：1-B——冻结 Control Plane 技术选型

## 项目定位

AgentDock 是一个多租户 Agent 运行时与安全执行平台。

核心链路：

用户提交 Agent Task
→ Go Control Plane
→ 调度与状态机
→ Redis 可靠队列
→ Docker / Kubernetes 沙箱
→ Python Agent Worker
→ MCP 工具调用
→ 日志、产物、结果与审计

## 已完成

- 明确项目名称、定位与核心链路
- 完成本地系统资源检查
- 打通 Docker Desktop 与 WSL 集成
- 验证 Docker Engine 和 Docker Compose
- 验证 K3s 单节点集群
- 建立 Python 3.12 独立 Worker 环境
- 安装 GCC 和 Go 竞态检测依赖
- 统一 kubectl 命令
- 初始化 Git 仓库
- 初始化 Go Module
- 建立项目基础目录
- 创建项目 README
- 创建 ADR-0001 初始架构决策记录
- 补齐工程目录占位文件
- 完成首次本地 Git 提交
- 创建公开 GitHub 远程仓库
- 建立 main 与 origin/main 跟踪关系
- 将工程基线成功推送至 GitHub

## 已接受架构决定

| 编号 | 架构决定 |
|---|---|
| AD-001 | 使用单仓库管理 Go、Python、部署、测试与文档 |
| AD-002 | Go 负责 Control Plane |
| AD-003 | Python 负责沙箱内 Agent 执行 |
| AD-004 | PostgreSQL 是任务持久化事实来源 |
| AD-005 | Redis 承担临时队列和调度状态 |
| AD-006 | Task 与 Attempt 分离 |
| AD-007 | 使用统一 Runtime Backend 接口 |
| AD-008 | 先实现 Docker，再实现 Kubernetes Job |
| AD-009 | Agent 运行环境默认采用最小权限 |

## 计划接口

当前尚未冻结代码接口。

后续计划定义：

- Task State Machine
- Task Repository
- Attempt Repository
- Reliable Queue
- Runtime Backend
- Audit Writer
- Artifact Store
- Tool Policy Evaluator

## 计划数据结构

- Tenant
- User
- Role
- Task
- TaskAttempt
- TaskEvent
- Artifact
- AuditLog
- MCPServer
- ToolPolicy

具体字段、索引和数据库约束尚未冻结。

## 阶段 1 已完成

- 冻结阶段 1 最小需求与接口契约
- 确定 Task 初始状态为 created
- 确定租户级幂等键语义
- 确定 Task 查询必须在数据库层携带 tenant_id
- 确定 live 与 ready 健康检查职责分离
- HTTP 选择 Go 标准库 net/http ServeMux
- 日志选择标准库 log/slog JSONHandler
- 配置采用自定义强类型环境变量解析
- PostgreSQL选择pgx/v5与pgxpool
- 数据访问阶段 1 采用手写SQL
- 数据库迁移选择Goose v3 SQL Migration
- 数据库迁移与应用启动生命周期分离

## 阶段 0 验收结果

- 本地系统：Ubuntu 24.04.1 on WSL2
- Go：1.25.0
- Python Worker：独立 Python 3.12.3 虚拟环境
- Docker Engine：29.5.2
- Docker Compose：5.1.3
- Kubernetes：K3s 1.35.5，节点 Ready
- Git 默认分支：main
- 初始提交：e2e6d40
- GitHub 仓库：wuge-xu/agentdock
- 仓库可见性：Public
- 远程同步状态：main 已跟踪 origin/main

阶段 0 已建立可复现的开发环境、清晰的工程边界、初始架构契约和持续 GitHub 展示链路。

## 当前未完成事项

1. 冻结阶段 1 最小需求范围
2. 确定 Go HTTP 路由方案
3. 建立 Control Plane 最小启动入口
4. 实现存活和就绪健康检查
5. 选择 PostgreSQL 驱动与迁移工具
6. 创建 PostgreSQL 与 Redis 本地基础设施
7. 定义 Task 与 TaskAttempt 数据结构
8. 定义第一版任务状态机
9. 实现 Task 创建与查询
10. 增加单元测试和集成测试

## 下一里程碑

启动 PostgreSQL
→ 启动 Go Control Plane
→ 调用健康检查接口
→ 创建 Task
→ PostgreSQL 保存 Task
→ 根据 Task ID 查询

## 风险与待确认事项

- PostgreSQL 与 Redis 之间的一致性方案尚未确定
- 任务取消与正常完成可能发生竞态
- Redis 数据丢失后的任务重建方式尚未设计
- Docker Socket 权限和安全边界需要单独设计
- Kubernetes Job 与 Task 状态的映射规则尚未确定
- MCP 工具授权和网络访问控制模型尚未确定
