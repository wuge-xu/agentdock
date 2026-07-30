# AgentDock

**多租户 Agent 运行时与安全执行平台**

AgentDock 是一个以 Go Control Plane 为核心的 Agent Runtime 平台，用于安全、可靠地调度和执行多租户 AI Agent 工作负载。

## 核心链路

~~~text
用户提交 Agent Task
→ Go Control Plane
→ 调度与状态机
→ Redis 可靠队列
→ Docker / Kubernetes 沙箱
→ Python Agent Worker
→ MCP 工具调用
→ 日志、产物、结果与审计
~~~

## 核心能力

- Go API与任务状态机
- PostgreSQL持久化
- Redis可靠队列与故障恢复
- Docker和Kubernetes隔离执行
- CPU、内存、超时与并发限制
- 取消、重试、幂等与死信处理
- 多租户与RBAC
- MCP Server注册与调用治理
- 工具白名单和网络访问控制
- 审计日志与产物管理
- Prometheus指标与故障演练

## 技术栈

- Go 1.25
- Python 3.12
- PostgreSQL
- Redis
- Docker
- Kubernetes
- MCP
- Prometheus
- Grafana

## 当前状态

阶段 0 工程基线与架构契约已完成，下一阶段将实现最小 Go Control Plane、健康检查与基础设施连接。

详细进度见 [`docs/PROJECT_STATUS.md`](docs/PROJECT_STATUS.md)。
