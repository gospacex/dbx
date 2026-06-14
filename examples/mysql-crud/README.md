# mysql-crud

`dbx` 框架的端到端示例：构建 MySQL 操作句柄、完成 5 个标准 CRUD 方法、并启用调用链记录 SQL 语句。

本示例作为独立 Go module 存在，通过 `replace` 指令链接到本地 `dbx` 工作区。

## 概述

本示例演示 4 个 dbx 入口与 4 个子命令的对应关系：

| 子命令 | dbx 入口 | 入口形态 |
|--------|----------|----------|
| `o`  | `dbsql.Open(cfg)`                | 硬编码 `*config.MySQLConfig` |
| `p`  | `dbsql.OpenPath(path)`           | 从 yaml 文件加载 |
| `oc` | `dbsql.OpenCluster(cc)`          | 硬编码 `*config.ClusterConfig`（含 read/write splitting） |
| `pc` | `dbsql.OpenClusterPath(path)`    | 从 yaml 文件加载 cluster 配置 |

每个子命令执行同一套 5 步 CRUD 序列：Create → Get → List → Update → Delete。

## 构建

```bash
cd examples/mysql-crud
go mod tidy
go build ./...
```

## 运行

```bash
# 1) 硬编码配置 + 单 MySQL
go run . o

# 2) yaml 配置 + 单 MySQL
cp mysql.example.yaml mysql.yaml
# 编辑 mysql.yaml 把 password: REPLACE_ME 改成实际密码
go run . p -config mysql.yaml

# 3) 硬编码配置 + cluster
go run . oc

# 4) yaml 配置 + cluster
cp cluster.example.yaml cluster.yaml
# 编辑 cluster.yaml 把 password: REPLACE_ME 改成实际密码
go run . pc -config cluster.yaml
```

## 测试命令

从 `examples/mysql-crud/` 目录运行（该示例有独立 `go.mod`）：

```bash
# 4 个子命令。.example.yaml 内密码已预置,直接走文件即可
# (等价于上一节"cp + 编辑 REPLACE_ME"的工作流)
go run . p -config mysql.example.yaml   # yaml + 单 MySQL(最常用)
go run . o                              # 硬编码 + 单 MySQL
go run . pc -config cluster.example.yaml
go run . oc

# 静态检查与格式化
go build ./...
go vet ./...
gofmt -l .                              # 无输出 = 全部 gofmt-clean
```

仓库根目录运行（覆盖 example module 引用到的 `orm` / `dbsql`）：

```bash
go test -count=1 -race ./orm/... ./dbsql/...
```

成功标志：`go run` 末尾打印 `[INFO] done`（exit 0）。trace 链路层面，任一 exporter
（jaeger UI / `redis-cli XLEN otel-traces` / `kcat -C`）应至少收到 6 个 span，共享同一
个 trace id——`user.crud` 父 + 5 个 `db.*` 子（create/query/query/update/delete）+ AutoMigrate
触发的 `db.query`。

## 配置文件结构

`mysql.example.yaml` 包含 4 个段：

| 段 | 字段 |
|----|------|
| `mysql:` | `host` / `port` / `username` / `password` / `database` / `charset` |
| `pool:`  | `max_open_conns` / `max_idle_conns` / `conn_max_lifetime` / `conn_max_idle_time` |
| `trace:` | `enabled` / `service_name` / `exporter` / `endpoint` / `protocol` / `sampler_type` / `sampler_ratio` |
| `logger:` | `level` / `color` / `slow_threshold_ms` / `ignore_record_not_found` |

> `trace:` 段字段名与 mqx 标准对齐（`service_name` / `protocol: http` / `sampler_type: always_on`），
> 内部通过 `mqxTraceToDBX` 翻译到 dbx 内部 `config.TracingConfig`。

`cluster.example.yaml` 与上同构，仅顶层段为 `cluster:`（含 `sources:` / `replicas:` / `load_balance`）。

## Trace Exporter 切换

3 种 exporter 全部经过 `dbsql.CreateExporter` 单一入口构造；切换只需改 `trace.exporter` 和 `trace.endpoint`。

### jaeger

```yaml
trace:
  enabled: true
  service_name: mysql-crud-example
  exporter: jaeger
  endpoint: localhost:4318
  protocol: http
  sampler_type: always_on
  sampler_ratio: 1.0
```

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

验证：浏览器打开 <http://localhost:16686>，Service 下拉应出现 `mysql-crud-example`。

### kafka

```yaml
trace:
  enabled: true
  service_name: mysql-crud-example
  exporter: kafka
  endpoint: localhost:9092
  topic: otel-traces
```

```bash
docker run -d --name kafka \
  -p 9092:9092 \
  bitnami/kafka:latest
```

验证：

```bash
kcat -b localhost:9092 -t otel-traces -C -e -o beginning
```

应看到至少 1 条 protobuf 编码的 span。

### redis_stream

```yaml
trace:
  enabled: true
  service_name: mysql-crud-example
  exporter: redis_stream
  endpoint: localhost:6379
  stream: otel-traces
```

```bash
docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine
```

验证：

```bash
redis-cli XLEN otel-traces
```

返回值应 ≥ 1。

## 关闭顺序

`db.go` 中 4 个 open 函数返回的 `shutdown` 闭包固定按以下顺序执行：

1. `TracerProvider.Shutdown(ctx)` — 停止接收新 span
2. `TracerProvider.ForceFlush(ctx)` — 强制 flush 残留 span
3. `sqlDB.Close()` — 关闭底层连接池

通过 `defer` 注册，panic 路径同样保证按此顺序。

## 排错

| 现象 | 原因 | 修复 |
|------|------|------|
| `[FATAL] config: password not set, replace REPLACE_ME in <file>` | 配置文件 password 仍为占位符 | 编辑文件替换 `REPLACE_ME` |
| `[FATAL] dbsql: tracing: ...` 且 exporter=kafka | broker 不可达 | 启动 `bitnami/kafka` 容器或调整 `endpoint` |
| `[FATAL] dbsql: tracing: ...` 且 exporter=redis_stream | redis 不可达 | 启动 `redis:7-alpine` 容器或调整 `endpoint` |
| `[FATAL] open: ...` 且 trace.enabled=true 但 jaeger 不可达 | 4318 端口无服务 | 启动 jaeger 容器，或将 `trace.enabled` 设为 `false` 走降级路径 |
| stdout 全无 `[SQL]` 行 | `logger.level: silent` | 改成 `info` 或 `warn` |

## 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功（含 `gorm.ErrRecordNotFound` 0-rows 软失败） |
| 2 | 配置/参数错误 |
| 3 | 连接错误（MySQL 不可达 / trace broker 不可达） |
| 4 | 业务错误（重复主键 / 约束冲突） |
