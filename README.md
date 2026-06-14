<div align="center">

  <h1>DBX</h1>
  <p><strong>一行代码,生产级 GORM 接入底座</strong></p>
  <p>专为 Go 微服务打造的企业级数据库连接基建、读写分离与可观测性方案。</p>

  [![Go Version](https://img.shields.io/badge/go-1.26%2B-blue.svg)](https://golang.org/dl/)
  [![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
  [![GORM V2](https://img.shields.io/badge/GORM-V2-orange.svg)](https://gorm.io)
  [![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44-blueviolet.svg)](https://opentelemetry.io)
</div>

---

## 项目定位

**DBX 不代理、不封装、不阉割 GORM 本身。** 当 `dbsql.Open*` 返回 `*gorm.DB` 时,调用方拿到的是 100% 原生 GORM 句柄 —— 事务、Preload、Scopes、Callbacks、Hooks 全部保留,无 `interface{}` 断言、无反射调用。

DBX 只做 4 件实事:

1. **配置层** —— 8 大数据库的 YAML/JSON/TOML 统一加载与校验
2. **连接层** —— DSN 构造、连接池调优、Fail-Fast
3. **集群层** —— 基于 `gorm.io/plugin/dbresolver` 的读写分离
4. **可观测层** —— OTel Trace 注入 + 3 路 Exporter(jaeger / kafka / redis_stream)

---

## 核心特性

| 特性 | 说明 |
| :--- | :--- |
| 8 大数据库 | MySQL / TiDB / MariaDB / PostgreSQL / GaussDB / MSSQL / Oracle / SQLite |
| 零 ORM 损耗 | 返回原生 `*gorm.DB`,无包装、无 interface 断言 |
| 强类型 API | `Open` 返回单库句柄,`OpenCluster` 返回带读写分离的句柄 |
| 多格式配置 | YAML / JSON / TOML,扩展名自动识别 |
| 读写分离 | 内置 `dbresolver`,支持 `round_robin` / `random` 策略 |
| OTel 原生 | jaeger OTLP HTTP + kafka + redis_stream,自动注入 GORM callback |
| 优雅关闭 | `tp.Shutdown` 排空 span + `sqlDB.Close()` 排空连接池,顺序固定 |

---

## 数据库矩阵

| 数据库 | config 类型 | 底层驱动 | 备注 |
| :--- | :--- | :--- | :--- |
| MySQL | `*config.MySQLConfig` | `gorm.io/driver/mysql` | — |
| TiDB | `*config.TiDBConfig` | `gorm.io/driver/mysql` | 协议兼容 MySQL |
| MariaDB | `*config.MariaDBConfig` | `gorm.io/driver/mysql` | 协议兼容 MySQL |
| PostgreSQL | `*config.PostgresConfig` | `gorm.io/driver/postgres` | — |
| GaussDB | `*config.GaussDBConfig` | `gorm.io/driver/postgres` | 协议兼容 PostgreSQL |
| MSSQL | `*config.MSSQLConfig` | `gorm.io/driver/sqlserver` | — |
| Oracle | `*config.OracleConfig` | `gorm.io/driver/oracle` | 底层 `godror`,需 CGO |
| SQLite | `*config.SQLiteConfig` | `gorm.io/driver/sqlite` | 纯文件型 |

> TiDB / MariaDB 复用 MySQL 驱动,无需额外 import。GaussDB 复用 PostgreSQL 驱动。

---

## 快速开始

### 安装

```bash
go get github.com/gospacex/dbx
```

### 方式一:Go struct 直传

```go
import (
    "github.com/gospacex/dbx/config"
    "github.com/gospacex/dbx/dbsql"
)

db, err := dbsql.Open(&config.MySQLConfig{
    Host:     "127.0.0.1",
    Port:     3306,
    Username: "app",
    Password: "secret",
    Database: "orders",
    Pool: &config.PoolConfig{
        MaxOpenConns:    50,
        MaxIdleConns:    10,
        ConnMaxLifetime: 1800,
    },
})
if err != nil {
    log.Fatal(err)
}

// 100% GORM 原生 API
var users []User
db.Where("name = ?", "alice").Find(&users)
```

### 方式二:YAML 文件

`db.yaml`:

```yaml
mysql:
  host: 127.0.0.1
  port: 3306
  username: app
  password: secret
  database: orders
  pool:
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: 1800
    conn_max_idle_time: 600
  trace:
    enabled: true
    service_name: order-service
    exporter: jaeger
    endpoint: localhost:4318
    protocol: http
    sampler_type: always_on
    sampler_ratio: 1.0
```

`main.go`:

```go
db, err := dbsql.OpenPath("db.yaml")
if err != nil {
    log.Fatal(err)
}
```

---

## API 矩阵(4 个入口)

| 场景 | struct 直传 | YAML 文件 |
| :--- | :--- | :--- |
| **单库** | `dbsql.Open(cfg config.DBConfig)` | `dbsql.OpenPath(path string)` |
| **集群(读写分离)** | `dbsql.OpenCluster(cc *config.ClusterConfig)` | `dbsql.OpenClusterPath(path string)` |

四个函数统一签名:

```go
func Open(cfg config.DBConfig) (*gorm.DB, error)
func OpenPath(path string) (*gorm.DB, error)
func OpenCluster(cc *config.ClusterConfig) (*gorm.DB, error)
func OpenClusterPath(path string) (*gorm.DB, error)
```

> 另有底层入口 `config.Load(path) (DBConfig, *TracingConfig, error)`,允许用户自行处理两份配置。

---

## 集群与读写分离

集群基于 `gorm.io/plugin/dbresolver` 实现。DBX 在配置层封装了 3 个关键字段:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `sources` | `[]NodeConfig` | 主库列表(写操作路由) |
| `replicas` | `[]NodeConfig` | 从库列表(读操作路由) |
| `load_balance` | `string` | `round_robin`(默认) / `random` |

`cluster.yaml`:

```yaml
cluster:
  driver: mysql
  load_balance: round_robin
  sources:
    - host: master.db
      port: 3306
      username: rw
      password: secret
      database: orders
  replicas:
    - host: slave1.db
      port: 3306
      username: ro
      password: secret
      database: orders
    - host: slave2.db
      port: 3306
      username: ro
      password: secret
      database: orders
  pool:
    max_open_conns: 200
    max_idle_conns: 40
  trace:
    enabled: true
    service_name: order-cluster
    exporter: jaeger
    endpoint: localhost:4318
```

```go
db, err := dbsql.OpenClusterPath("cluster.yaml")
if err != nil {
    log.Fatal(err)
}

db.Create(&user)   // → master
db.Find(&users)    // → slave1 / slave2(round_robin)
```

---

## 可观测性:OTel Trace 注入

DBX 在三个层级注入 OTel:

1. **应用层**: 业务代码可显式 `tracer.Start(ctx, "name")`,span 沿 context 传递
2. **GORM 层**: `orm.Open` 注册 v2 callback,所有 `db.create/query/update/delete` 自动产生 `SpanKind=Client` 的 span
3. **Exporter 层**: 3 路 exporter 任选其一,统一通过 `trace:` 段配置

### Trace 配置规范(mqx 对齐)

DBX 的 `trace:` 段字段名与 mqx 标准 1:1 对齐,跨语言团队可复用同一份配置:

| 字段 | 含义 | 取值 |
| :--- | :--- | :--- |
| `enabled` | 是否启用追踪 | `true` / `false` |
| `service_name` | 服务名(jaeger UI 显示) | 任意字符串 |
| `exporter` | 导出器 | `jaeger` / `kafka` / `redis_stream` |
| `endpoint` | exporter endpoint | host:port / 逗号分隔 |
| `protocol` | OTel 协议(仅 jaeger) | `http` |
| `sampler_type` | 采样器类型 | `always_on` / `parentbased_always_on` |
| `sampler_ratio` | 采样率 | `0.0` ~ `1.0` |
| `mode` | 拓扑模式 | `single` / `cluster`(仅 redis_stream) |

> 字段名严格遵循 mqx:`service_name`(非 `service`)、`protocol: http`(内部映射到 `http/protobuf`)、`sampler_type: always_on`(内部映射到 `parentbased_always_on`)。这层翻译由 `dbsql` 完成,用户无需关心 OTel SDK 细节。

### 3 路 Exporter 切换

#### jaeger(OTLP HTTP)

```yaml
trace:
  enabled: true
  service_name: my-service
  exporter: jaeger
  endpoint: localhost:4318
  protocol: http
  sampler_type: always_on
  sampler_ratio: 1.0
```

```bash
docker run -d -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one:latest
# 浏览器访问 http://localhost:16686,Service 下拉选 my-service
```

#### kafka

```yaml
trace:
  enabled: true
  service_name: my-service
  exporter: kafka
  endpoint: localhost:9092                # 多 broker 用逗号分隔
  topic: otel-traces
  kafka_security_protocol: PLAINTEXT
  kafka_sasl_mechanism: PLAIN
  kafka_username: ""
  kafka_password: ""
```

> 内部走 `kafkax.POS`(多 broker 通过 `Addrs` 数组支持),`mode` 字段被忽略。

#### redis_stream

```yaml
trace:
  enabled: true
  service_name: my-service
  exporter: redis_stream
  endpoint: localhost:6379                # 集群模式: 逗号分隔多个节点
  mode: single                            # single → redisx.POS | cluster → redisx.POC
  stream: otel-traces
  redis_password: ""
```

```bash
redis-cli XLEN otel-traces   # 返回值 ≥ 1 表示有 span 进入
```

### Exporter 分发优先级

`observability/tracing.MQXSpanExporter` 是统一分发器,内部按 `kafka > redis > jaeger` 优先级写入:

```text
   Span buffer
       │
       ▼
 MQXSpanExporter.ExportSpans
       │
       ├── kafka  enabled? → KafkaExporter
       ├── redis  enabled? → RedisExporter
       └── jaeger enabled? → jaeger OTLP HTTP
```

任一路 exporter 故障不影响其它两路,默认所有失败的 span 走 noop(不阻塞业务)。

---

## 优雅关闭

退出时按以下顺序关闭,确保不丢 span:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 1. 停止 OTel 接收新 span
// 2. 强制 flush 残留 span 到 exporter
// (TP 的 Shutdown 内部已含 ForceFlush)
tp.Shutdown(ctx)

// 3. 关闭连接池
sqlDB, _ := db.DB()
sqlDB.Close()
```

参考 `examples/mysql-crud/main.go` 中 `defer shutdown(ctx)` 闭包的标准实现。

---

## 错误处理与 DSN 脱敏

```go
import dbx "github.com/gospacex/dbx"

// 哨兵错误,errors.Is 判定
dbx.ErrInvalidConfig
dbx.ErrDriverUnsupported

// DSN 脱敏:把 password 段替换为 xxxxx
safe := dbx.RedactDSN("app:secret@tcp(127.0.0.1:3306)/orders")
// → "app:xxxxx@tcp(127.0.0.1:3306)/orders"
```

所有错误都使用 `fmt.Errorf("...: %w", err)` 包装,保留 `errors.Is / errors.As` 链路。

---

## 示例

| 示例 | 路径 | 演示内容 |
| :--- | :--- | :--- |
| mysql-crud | [`examples/mysql-crud/`](examples/mysql-crud/) | 4 个 dbx 入口 + 5 步 CRUD + 3 路 trace exporter |

```bash
cd examples/mysql-crud
go run . p  -config mysql.example.yaml   # yaml + 单 MySQL
go run . o                                # 硬编码 + 单 MySQL
go run . pc -config cluster.example.yaml  # yaml + 集群
go run . oc                               # 硬编码 + 集群
```

每个子命令执行同一套 5 步 CRUD 序列:Create → Get → List → Update → Delete,全部串在父 span `user.crud` 下,便于在 jaeger UI 看到一条完整的调用树。

---

## 测试与质量

```bash
# 全部包
go test -count=1 -race ./orm/... ./dbsql/...

# 覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# E2E(需先起 docker-compose,见 e2e/db_e2e_test.go:4-13)
go test -tags e2e -count=1 -timeout 60s ./e2e/...
```

CI 规范:所有改动须 `go vet` + `gofmt -l` 干净、`go test -race` 全过。

---

## 项目结构

```
dbx/
├── config/                    # 统一配置层
│   ├── config.go              # DBConfig 接口 & BaseDBConfig 基类
│   ├── loader.go              # 8 路 LoadXxx (YAML/JSON/TOML 自动识别)
│   ├── cluster.go             # ClusterConfig & NodeConfig (读写分离)
│   ├── pool.go                # PoolConfig (连接池调优)
│   ├── tracing.go             # TracingConfig (内部 dbx 形态)
│   ├── mysql.go / postgres.go / tidb.go / mariadb.go /
│   │   gaussdb.go / mssql.go / oracle.go / sqlite.go
│   │                           # 8 大数据库独立 config (Validate + DSN)
│   └── *_test.go
├── dbsql/                     # 入口层
│   ├── dbsql.go               # Open / OpenPath — 单库
│   ├── cluster.go             # OpenCluster / OpenClusterPath — 集群
│   ├── tracer.go              # OTel Exporter 构造 + GORM 注入
│   └── *_test.go
├── orm/                       # GORM 驱动分发 + Trace Callback
│   ├── gorm.go                # Dialector 路由
│   ├── gorm_tracing.go        # GORM v2 callback 安装
│   └── *_test.go
├── observability/tracing/     # 多后端 OTel Exporter
│   ├── exporter.go            # MQXSpanExporter — 优先级: kafka > redis > jaeger
│   ├── kafka_exporter.go
│   ├── redis_exporter.go
│   └── helpers.go
├── examples/mysql-crud/       # 端到端 demo (4 subcommand × 3 exporter)
├── e2e/                       # build tag e2e 的集成测试
├── errors.go                  # 哨兵错误 + RedactDSN
├── PRD.md                     # 产品需求
├── openspec/                  # 规格与变更管理
└── docs/                      # 设计与决策文档
```

---

## 主要依赖

| 类别 | 依赖 | 用途 |
| :--- | :--- | :--- |
| ORM | `gorm.io/gorm v1.25.12` | 主 ORM |
| 驱动 | `gorm.io/driver/{mysql,postgres,sqlite,sqlserver}` | 8 大数据库驱动 |
| 集群 | `gorm.io/plugin/dbresolver v1.5.3` | 读写分离 |
| MQ 适配 | `github.com/gospacex/mqx`(本地 replace) | kafka / redis 连接池 |
| 追踪 | `go.opentelemetry.io/otel/sdk v1.44.0` | OTel SDK |
| OTLP HTTP | `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0` | jaeger 导出 |
| 配置 | `gopkg.in/yaml.v3` | YAML 解析 |
|  | `github.com/BurntSushi/toml` | TOML 解析 |

---

## 许可证

MIT License —— 详见 [LICENSE](LICENSE)。
