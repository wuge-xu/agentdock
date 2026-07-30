# AgentDock 项目状态

- 最后更新：2026-07-29
- 当前阶段：阶段 0——工程基线与架构契约
- 当前步骤：0-E——仓库骨架与架构记录

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

## 当前未完成事项

1. 创建初始架构 ADR
2. 补齐空目录占位文件
3. 验证仓库工具命令
4. 完成首次 Git 提交
5. 确定阶段 1 最小需求
6. 定义任务状态机
7. 定义 Task 和 Attempt 数据结构
8. 选择 PostgreSQL 驱动与迁移工具
9. 创建 PostgreSQL 和 Redis 本地环境
10. 实现首个健康检查接口

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
