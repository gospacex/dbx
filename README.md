# DBX

<div align="center">

**企业级 Go 数据库接入底座 —— 统一配置、连接池治理、读写分离与全链路可观测**

[![Go](https://img.shields.io/badge/go-1.26.2+-00ADD8.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GORM](https://img.shields.io/badge/GORM-v2-orange.svg)](https://gorm.io)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44-blueviolet.svg)](https://opentelemetry.io)
[![Build](https://img.shields.io/badge/build%20%2B%20vet-passing-brightgreen.svg)](#质量门禁)

</div>

---

## 1. 定位与设计边界

DBX 不做 ORM，也不包装 GORM。所有 `dbsql.Open*` 返回的是**原生 `*gorm.DB`** —— 事务、`Preload`、`Scopes`、自定义 Callbacks、Hooks 全部保留，没有 `interface{}` 断言，没有反射代理。

DBX 只负责 ORM 之外、但每个服务都要重写一遍的那一层：

| 能力层 | 职责 | 载体包 |
| :--- | :--- | :--- |
| 配置层 | 8 大数据库的 YAML / JSON / TOML 统一加载、默认值填充、校验 | `config` |
| 连接层 | DSN 构造、连接池调优、Fail-Fast 校验 | `dbsql` `orm` |
| 集群层 | 基于 `dbresolver` 的读写分离 | `dbsql` `config` |
| 可观测层 | OTel Span 注入 + jaeger / kafka / redis_stream 三路导出 | `orm` `observability/tracing` |
| 日志层 | logx 适配的 GORM Logger（慢查询 + 链路关联） | `orm` |
| 注册层 | hubx.ClientProvider 实例工厂 | `hubx/<driver>x` |

**非目标**（明确不做，避免误用）：

- 不提供查询构造器、Unit of Work、Repository 抽象 —— 那是 GORM 的职责
- 不做后台健康检查 pinger —— `health_check` 只读入配置，实际由调用方 `PingContext`
- 不做逐语句重试 —— 重试属服务层语义（`FEATURE_MATRIX.md` 中标记为预留）

---

## 2. 快速开始

### 2.1 安装

```bash
go get github.com/gospacex/dbx
```

> ⚠️ **仓库形态说明**：本模块依赖 `configx` / `logx` / `hubx` 三个**未发布**的兄弟模块，通过 `replace` 指向本地路径。Go 的 `replace` **不会传递给下游 Module**，因此任何消费方都必须在自己的 `go.mod` 里重复声明这些 replace，否则构建无法解析依赖。详见 [§10 已知约束](#10-已知约束)。

### 2.2 方式一：结构体直传

```go
import (
    "github.com/gospacex/dbx/config"
    "github.com/gospacex/dbx/dbsql"
)

db, err := dbsql.Open(&config.MySQLConfig{
    BaseDBConfig: config.BaseDBConfig{
        CommonNetworkConfig: config.CommonNetworkConfig{
            Host:     "127.0.0.1",
            Port:     3306,
            Username: "app",
            Password: "secret",
            Database: "orders",
        },
        Pool: &config.PoolConfig{
            MaxOpenConns:    50,
            MaxIdleConns:    10,
            ConnMaxLifetime: 1800,
        },
    },
})
if err != nil {
    log.Fatal(err)
}

// 100% 原生 GORM API
var users []User
db.Where("name = ?", "alice").Find(&users)
```

### 2.3 方式二：配置文件

`db.yaml`：

```yaml
mysql:
  host: 127.0.0.1
  port: 3306
  username: app
  password: secret
  database: orders
  charset: utf8mb4
  # pool 必须嵌套在驱动段内部
  pool:
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: 1800
    conn_max_idle_time: 600

# tracing 必须是顶层段 —— 注意键名是 tracing，不是 trace
tracing:
  enabled: true
  service: order-service
  exporter: jaeger
  endpoint: localhost:4318
  sampler_type: parentbased_always_on
  sampler_ratio: 1.0
```

```go
db, err := dbsql.OpenPath("db.yaml")
```

> 🔴 **本节是本项目最容易踩的坑，请务必照抄上面的层级结构。** `config.Load` 只识别**顶层** `tracing:` 段，且 `pool:` 必须**嵌套在驱动段内**。写成顶层 `trace:` 或顶层 `pool:` 时 yaml 解析不报错，配置会被**静默丢弃** —— 连接池退化成默认值、链路追踪完全不上报。详见 [§5.3](#53-追踪配置的三个陷阱)。

### 2.4 方式三：configx 配置中心（生产推荐）

读取 `xmiddleware.sqlx` 契约，零 YAML 文件、支持热线更新。

**统一入口三件套**（与其余 x 系列 SDK 同签名，`T` = `*gorm.DB`）：

```go
import (
    "github.com/gospacex/configx"
    "github.com/gospacex/dbx/dbsql"
)

// 一步到位：读配置 → 打开 xmiddleware.sqlx.databases.default
db, err := dbsql.ReadAndCreate(configx.ConfigOptions{ConfigFile: "config.yaml"})

// 失败即退出进程（等价上句 + os.Exit(1)）
db := dbsql.MustReadAndCreate(configx.ConfigOptions{})

// 只解析配置、不开连接；参数预检用
err := dbsql.ReadConfig(configx.ConfigOptions{ConfigFile: "config.yaml"})
```

`ReadAndCreate` 固定打开 **`default`** 实例（与 `lockx` / `objectx` 的"单资源"约定一致）。
需要打开其它实例（读写分离的 `replica` 等）或自行提供 logger 时，用底层两件：

```go
db, err := dbsql.OpenFromConfigx(cfg, "replica", logx.GetGlobal())
sc := dbsql.ReadConfigFromConfigx(cfg, "replica")   // 只要配置不要连接
```

```yaml
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      default:
        dsn: "app:secret@tcp(127.0.0.1:3306)/orders?parseTime=true"
        dialect: mysql          # 可选，缺省从 DSN 推断
        max_open_conns: 50
        max_idle_conns: 10
        conn_max_lifetime: 5m   # duration 字符串，0 = 不限制
        slow_threshold: 200ms
        enable_log: true
        log_level: warn
        enable_trace: true
        tracing:                # 子段优先于 enable_trace
          exporter: jaeger
          endpoint: localhost:4318
          service: order-service
```

> 用 `ReadAndCreate` / `MustReadAndCreate` 时，若配置里 `enable_trace: true`，
> 函数会装配全局 TracerProvider，调用方仍需在进程退出前
> `defer dbsql.ShutdownTracerProvider(ctx)` 冲刷 span。

---

## 3. 入口 API 矩阵

**配置驱动入口（统一三件套）**

| 函数 | 签名 | 说明 |
| :--- | :--- | :--- |
| `ReadConfig` | `func ReadConfig(opts ...configx.ConfigOptions) error` | 只解析配置，不开连接 |
| `ReadAndCreate` | `func ReadAndCreate(opts ...configx.ConfigOptions) (*gorm.DB, error)` | 读配置并打开 `default` 实例 |
| `MustReadAndCreate` | `func MustReadAndCreate(opts ...configx.ConfigOptions) *gorm.DB` | 同上，失败写 stderr 并 `os.Exit(1)` |
| `ReadConfigFromConfigx` | `func ReadConfigFromConfigx(cfg configx.Config, name string) *SQLConfig` | 从已有句柄取 `xmiddleware.sqlx.databases.<name>`，不建连接，永不失败 |
| `OpenFromConfigx` | `func OpenFromConfigx(cfg configx.Config, name string, l logx.Logger) (*gorm.DB, error)` | 打开指定实例；`l` 可为 `nil`（回落到 `logx.GetGlobal()`） |

**三种配置来源 × 打开方式**

| 场景 | 结构体直传 | 配置文件 | configx 配置中心 |
| :--- | :--- | :--- | :--- |
| 单库 | `dbsql.Open(config.DBConfig)` | `dbsql.OpenPath(path)` | `dbsql.ReadAndCreate(opts...)` / `dbsql.OpenFromConfigx(cfg, name, logger)` |
| 集群（读写分离） | `dbsql.OpenCluster(*config.ClusterConfig)` | `dbsql.OpenClusterPath(path)` | — |

统一签名：

```go
func Open(cfg config.DBConfig) (*gorm.DB, error)
func OpenPath(path string) (*gorm.DB, error)
func OpenCluster(cc *config.ClusterConfig) (*gorm.DB, error)
func OpenClusterPath(path string) (*gorm.DB, error)
func OpenFromConfigx(cfg configx.Config, name string, l logx.Logger) (*gorm.DB, error)
```

底层差异化入口（自行接管配置与追踪两份对象时使用）：

```go
// 文件加载：自动识别驱动，返回 DBConfig + 顶层 TracingConfig
func config.Load(path string) (DBConfig, *TracingConfig, error)
func config.LoadMySQL(path string) (*MySQLConfig, *TracingConfig, error)   // 其余 7 个驱动同构
func config.LoadCluster(path string) (*ClusterConfig, *TracingConfig, error)

// configx 加载：driver 字符串分派
func config.LoadDBConfigFromConfigx(cc configx.Config, driver, name string) (DBConfig, *TracingConfig, error)
func config.LoadMySQLFromConfigx(cc configx.Config, name string) (*MySQLConfig, *TracingConfig, error)
```

**文件路径（`.yaml` / `.yml` / `.json` / `.toml`，按扩展名识别）**：若文件主名匹配驱动名（含别名 `postgresql`），直接分派；否则按文件中**第一个非空段**依次探测 `mysql → postgres → tidb → mariadb → gaussdb → mssql → oracle → sqlite`。

---

## 4. 数据库矩阵

| 数据库 | config 类型 | GORM Dialector | 集群 | hubx provider |
| :--- | :--- | :--- | :--- | :--- |
| MySQL | `*config.MySQLConfig` | `mysql` | ✅ | `hubx/mysqlx` (`dbx.mysql`) |
| PostgreSQL | `*config.PostgresConfig` | `postgres` | ✅ | `hubx/postgresx` (`dbx.postgres`) |
| SQLite | `*config.SQLiteConfig` | `sqlite` | ✅ | `hubx/sqlitex` (`dbx.sqlite`) |
| TiDB | `*config.TiDBConfig` | `mysql`（线协议兼容） | ✅ | `hubx/tidbx` (`dbx.tidb`) |
| MariaDB | `*config.MariaDBConfig` | `mysql`（线协议兼容） | ✅ | — |
| GaussDB | `*config.GaussDBConfig` | `postgres`（线协议兼容） | ✅ | — |
| MSSQL | `*config.MSSQLConfig` | `sqlserver` | ✅ | — |
| Oracle | `*config.OracleConfig` | ❌ **不支持** | ❌ | — |

> **Oracle 现状**：`orm.Dialector("oracle", dsn)` 返回显式错误 `"gorm: oracle driver requires CGo (godror) or pure-Go driver"`。配置类型保留以便契约兼容，但**不可用于建连**。
>
> ⚠️ `orm.SupportedDrivers()` 返回 8 个名字，其中含 `oracle` —— 该函数列出的是**配置层**支持的方言，不等于 `Dialector` 全部可建连。

**默认端口**：MySQL/TiDB/MariaDB `3306`、PostgreSQL/GaussDB `5432`、MSSQL `1433`、Oracle `1521`。

---

## 5. 配置参考

### 5.1 连接池（`config.PoolConfig`）

```yaml
pool:
  max_open_conns: 50       # 默认 50
  max_idle_conns: 10       # 默认 10，且自动封顶到 max_open_conns
  conn_max_lifetime: 1800  # 秒，默认 1800
  conn_max_idle_time: 600  # 秒，默认 600
```

> **语义差异**：文件路径的 `PoolConfig` 字段是**整数秒**，且 `<= 0` 一律替换为上表默认值，因此无法表达 `database/sql` 的 `0 = 无限制`；而 configx 路径（`OpenFromConfigx`）直接把 `time.Duration` 透传给 `database/sql`，`0` 即无限制。两条路径语义不同，跨入口迁移时需留意。

### 5.2 追踪配置（`config.TracingConfig`）

| 键 | 类型 | 默认 | 说明 |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | `false` | 关闭时不初始化任何 TracerProvider 设施 |
| `service` | string | `dbx` | OTel resource 的 `service.name` |
| `exporter` | string | `jaeger` | `jaeger` \| `kafka` \| `redis_stream` |
| `endpoint` | string | 按 exporter | `localhost:4318` / `localhost:9092` / `localhost:6379`；kafka 与 redis 支持逗号分隔多节点 |
| `sampler_type` | string | `parentbased_always_on` | 见下表 |
| `sampler_ratio` | float | `1.0` | 仅 ratio 采样器生效 |
| `mode` | string | `single` | `single` \| `cluster`；**仅 `redis_stream` 分支使用** |
| `topic` | string | `otel-traces` | kafka 专有 |
| `stream` | string | `otel-traces` | redis 专有 |
| `kafka_username` / `kafka_password` | string | — | kafka SASL |
| `kafka_security_protocol` / `kafka_sasl_mechanism` | string | — | kafka SASL |
| `redis_password` | string | — | redis 认证 |

**采样器取值**

| `sampler_type` | 行为 |
| :--- | :--- |
| `parentbased_always_on`、`always_on`、空串 | `ParentBased(AlwaysSample())` |
| `parentbased_always_off`、`always_off` | `ParentBased(NeverSample())` |
| `parentbased_traceidratio`、`traceidratio` | `ParentBased(TraceIDRatioBased(ratio))` |

> **已移除 `protocol` 键。** Jaeger 的 OTLP 传输方式不再由配置项指定，而是由 `endpoint` 的 scheme 推导：`host:port` 或 `http://…` → 明文 HTTP；`https://…` → TLS。老配置里的 `protocol: http` 会被忽略（不报错）。

### 5.3 追踪配置的三个陷阱

1. **层级**：文件路径只读取**顶层** `tracing:` 段。写在驱动段内（如 `mysql.tracing`）会被 `Validate()` 校验并填充默认值，但随后**被丢弃** —— 因为 `OpenPath` 只消费 `config.Load` 返回的顶层 `*TracingConfig`。结构体路径则相反：`dbsql.Open` 消费 `cfg.GetTracing()`。
2. **键名**：是 `tracing`（不是 `trace`），是 `service`（不是 `service_name`）。写错不报错，静默失效。
3. **Provider 覆盖**：`dbsql.ExtractTracingAndApply` 的幂等判断只覆盖它自己创建的 Provider，`otel.SetTracerProvider` 直接安装的外部 Provider **检测不到**。宿主若自建 Provider 且同时启用 dbx 追踪，dbx 会用新 Provider 覆盖全局，第一个 Provider 泄漏。要么让 dbx 全权管理，要么关闭 `enabled` 自行注入回调。

### 5.4 TracerProvider 生命周期

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 由 dbx 创建的全局 Provider，必须显式关闭以排空 span
defer func() {
    if err := dbsql.ShutdownTracerProvider(ctx); err != nil {
        log.Printf("tracer shutdown: %v", err)
    }
}()

sqlDB, _ := db.DB()
defer sqlDB.Close()
```

关闭顺序固定：**先 flush 追踪，再关连接池**。`ShutdownTracerProvider` 幂等、未初始化时为 no-op。

---

## 6. 可观测性

### 6.1 Span 注入机制

GORM v2 不会自动拾取 `otel.GetTracerProvider()`，dbx 自建最小回调集：`orm.Open` 为 create / query / update / delete 四组回调注册 before/after 钩子。

| 属性 | 值 |
| :--- | :--- |
| Span 名 | `db.<operation>` |
| SpanKind | `client` |
| Tracer 名 | `github.com/gospacex/dbx/orm` |
| 属性 | `db.operation`、`db.system`、`db.sql.table`、`db.query.text`（预编译 SQL，参数不内联） |
| 错误判定 | `gorm.ErrRecordNotFound` 视为正常 miss，不标记 span error；其余记 `RecordError` + `Status=Error` |

未提供 TracerProvider 时 `otel.Tracer` 返回 no-op tracer，回调零成本，可无条件调用。

### 6.2 三路 Exporter

`observability/tracing.MQXSpanExporter` 按 `kafka > redis > jaeger` 优先级分发：

```text
BatchSpanProcessor
        │
        ▼
MQXSpanExporter.ExportSpans
        ├── kafka  != nil → KafkaExporter    (kafkax.POS → Produce)
        ├── redis  != nil → RedisExporter    (redisx.POS/POC → XADD)
        └── jaeger != nil → OTLP over HTTP   (otlptracehttp)
```

**Shutdown 语义**：kafka 以 10s 超时 `Flush` 生产者队列；redis 为 no-op（连接生命周期委托 mqx）；jaeger 走 `otlptracehttp.Shutdown`。

```yaml
# jaeger —— 容器起法
tracing:
  enabled: true
  service: order-service
  exporter: jaeger
  endpoint: localhost:4318
```
```bash
docker run -d -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one:latest
# UI: http://localhost:16686 ，Service 选 order-service
```

```yaml
# kafka —— 多 broker 用逗号分隔；mode 对 kafka 无意义
tracing:
  enabled: true
  exporter: kafka
  endpoint: broker1:9092,broker2:9092
  topic: otel-traces
  mode: single
```

```yaml
# redis_stream —— mode 决定 single(redisx.POS) / cluster(redisx.POC)
tracing:
  enabled: true
  exporter: redis_stream
  endpoint: node1:6379,node2:6379
  mode: cluster
  stream: otel-traces
```
```bash
redis-cli XLEN otel-traces   # ≥1 表示 span 已入流
```

### 6.3 日志（`orm.WithLogxLogger`）

logx 版 GORM Logger，输出字段 `db.statement` / `db.elapsed` / `db.rows`；慢查询命中 `slow_threshold` 时降级为 `warn`。日志走 logx 的 `*Context` 方法，当 statement context 携带活动 span 时自动与调用链关联。`logger` 传 `nil` 会回退 `logx.GetGlobal()`（未注册 provider 时为安全 no-op）。

---

## 7. 集群与读写分离

基于 `gorm.io/plugin/dbresolver`：`sources` 承载写、`replicas` 承载读。

```yaml
cluster:
  driver: mysql
  load_balance: round_robin   # round_robin（默认） | random
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
  pool:
    max_open_conns: 200
    max_idle_conns: 40
```

```go
db, err := dbsql.OpenClusterPath("cluster.yaml")

db.Create(&user)   // → master
db.Find(&users)    // → slave1 / slave2（round_robin）
```

**校验规则**：`driver` 须属于集群白名单；`sources` 至少 1 个；`replicas` 可空；`load_balance` 非法值直接报错；`pool` 缺失时由 `Validate()` 补齐默认值。

---

## 8. hubx 注册中心 provider

四个 provider 均实现 `hubx.ClientProvider`，供运行时按实例名批量构建 client：

```go
import "github.com/gospacex/dbx/hubx/mysqlx"

provider := mysqlx.New()   // Name() == "dbx.mysql"
client, err := provider.Build("default", map[string]any{
    "config": map[string]any{
        "dsn":               "app:secret@tcp(127.0.0.1:3306)/orders?parseTime=true",
        "max_open_conns":    50,
        "max_idle_conns":    10,
        "conn_max_lifetime": "30m",   // 必须是时长字符串
    },
})
defer client.Close()
_ = client.HealthCheck(ctx)   // 内部 PingContext
```

> ⚠️ **`conn_max_lifetime` 的单位是纳秒语义，务必写成时长字符串。** 该字段类型为 `time.Duration`：
> - 写 `"30m"` / `"1h30m"` → 正确（依赖 `mapstructure.StringToTimeDurationHookFunc`，已内置）
> - 写裸整数 `1800` → 解析为 **1800 纳秒**（≈1.8µs），连接池中的连接会被立即判为过期
>
> 它与文件路径 `PoolConfig.ConnMaxLifetime`（整数秒）单位不同，切勿混用。

| 注册名 | 包 | 建连方式 |
| :--- | :--- | :--- |
| `dbx.mysql` | `hubx/mysqlx` | `orm.Open`（GORM 句柄 + 底层 `*sql.DB`） |
| `dbx.postgres` | `hubx/postgresx` | `sql.Open("postgres", dsn)` |
| `hubx/sqlitex` → `dbx.sqlite` | `hubx/sqlitex` | `sql.Open("sqlite3", dsn)` |
| `dbx.tidb` | `hubx/tidbx` | `sql.Open("mysql", dsn)` |

> decode 使用 `mapstructure` 且开启 `ErrorUnused`，YAML 键名拼错（如 `dsnn`）会立即报错而非静默忽略。
>
> ⚠️ `mysqlx` 的 `DB()` / `GORM()` 访问器定义在未导出类型 `client` 上，而 `Build` 返回的是 `hubx.Client` 接口 —— 外部调用方无法到达这两个方法。如需暴露 GORM 句柄，应改为导出 `client` 类型。

---

## 9. 错误处理

DBX 不使用哨兵错误变量，全程 `%w` 包装并加前缀，便于定位来源：

| 前缀 | 来源 | 示例 |
| :--- | :--- | :--- |
| `mysql: / postgres: / …` | `config` 校验 | `mysql: host and username are required` |
| `cluster:` | `ClusterConfig.Validate` | `cluster: unsupported driver "xxx"` |
| `tracing:` | `TracingConfig.Validate` | `tracing: unsupported exporter "zipkin"` |
| `dbsql:` | 入口层 | `dbsql: open: gorm: unsupported driver "oracle"` |
| `gorm:` | `orm.Dialector` | `gorm: oracle driver requires CGo (godror)` |
| `loader:` | 文件加载 | `loader: missing mysql section in db.yaml` |
| hubx `ErrConfigInvalid` / `ErrBuildFailed` | provider 构建 | `dbx.mysql/default: dsn is required` |

> ⚠️ **DSN 可能进入错误信息**（如驱动层返回的建连错误）。日志采集链路建议自行脱敏后再落盘。
>
> 已移除的旧导出符号：`dbx.ErrInvalidConfig`、`dbx.ErrDriverUnsupported`、`dbx.RedactDSN`；根包 `github.com/gospacex/dbx` 随之不再存在，请直接使用 `…/dbx/config`、`…/dbx/dbsql` 等子包。

---

## 10. 已知约束

阅读本节可以避免绝大多数集成故障。

1. **`replace` 不向下传递。** dbx 自己用 replace 解析 `configx` / `logx` / `hubx`，但这对下游 Module 无效。消费方必须在自己的 `go.mod` 重复声明：
   ```go
   replace github.com/gospacex/configx => <path>/configx
   replace github.com/gospacex/logx    => <path>/logx
   replace github.com/gospacex/hubx    => <path>/hubx
   ```
2. **不要把 `mqx` replace 到兄弟目录。** dbx 导入的是 upstream `v0.0.1` 的 `mqx/kafkax` 与 `mqx/redisx`；同仓库的 `../mqx` 兄弟模块包名布局不同（暴露 `kafka/` 而非 `kafkax/`），replace 过去会导致导入图断裂。这也曾使 `examples/mysql-crud` 长期无法构建，已在本次评审中修复。
3. **文件路径的追踪段必须是顶层 `tracing:`**（见 [§5.3](#53-追踪配置的三个陷阱)）。
4. **自建 TracerProvider 时勿同时开启 `enabled`**，否则 dbx 会覆盖全局 Provider（见 [§5.3](#53-追踪配置的三个陷阱)）。
5. **`examples/mysql-crud` 使用自有 YAML schema**（顶层 `pool:` / `trace:`，由 `loader.go` 自行解析并调用 `dbsql.Open`），它**不走** `dbsql.OpenPath`。该示例的 schema **不是** dbx 文件协议，勿作为配置范例参考。
6. **Oracle 不可建连**（见 [§4](#4-数据库矩阵)）。
7. **`hubx/*` provider 的编译依赖兄弟 `hubx` 模块**；该目录缺失时 `./hubx/...` 无法在此工作区构建，需先恢复 `replace` 目标。

---

## 11. 项目结构

```
dbx/
├── config/                     # 统一配置层
│   ├── config.go               # DBConfig 接口、CommonNetworkConfig、BaseDBConfig
│   ├── loader.go               # Load / LoadXxx：8 路驱动 + 格式/文件名分派
│   ├── configx_loader.go       # LoadDBConfigFromConfigx / TracingFromConfigx
│   ├── cluster.go              # ClusterConfig、NodeConfig（读写分离）
│   ├── pool.go                 # PoolConfig（连接池默认值与封顶）
│   ├── tracing.go              # TracingConfig（Exporter 常量、采样、默认值）
│   ├── {mysql,postgres,tidb,mariadb,gaussdb,mssql,oracle,sqlite}.go
│   └── *_test.go
├── dbsql/                      # 入口层
│   ├── dbsql.go                # Open / OpenPath
│   ├── cluster.go              # OpenCluster / OpenClusterPath
│   ├── configx.go              # ReadConfigFromConfigx / OpenFromConfigx
│   └── tracer.go               # CreateExporter / ExtractTracingAndApply / ShutdownTracerProvider
├── orm/                        # GORM 驱动分发 + 回调
│   ├── gorm.go                 # Open / OpenNoTrace / Dialector / SupportedDrivers
│   ├── gorm_tracing.go         # 4 组回调的 OTel span 注入
│   └── gorm_logx.go            # logx 版 GORM Logger
├── observability/tracing/      # 多后端 Exporter
│   ├── exporter.go             # MQXSpanExporter（优先级 kafka > redis > jaeger）
│   ├── kafka_exporter.go
│   ├── redis_exporter.go
│   └── helpers.go              # span → JSON 序列化
├── hubx/                       # hubx 注册中心 provider
│   └── {mysqlx,postgresx,sqlitex,tidbx}/
├── e2e/                        # `//go:build e2e` 集成测试（需 docker）
├── examples/
│   ├── mysql-crud/             # 端到端 demo（独立 module）
│   └── sqlite-configx/         # configx 契约 demo（独立 module）
├── FEATURE_MATRIX.md           # 功能 × 测试覆盖率对照
├── PRD.md
└── AGENTS.md
```

---

## 12. 质量门禁

```bash
# 静态检查（CI 要求全部干净）
gofmt -l .            # 无输出
go vet ./...

# 单元测试
go test -count=1 -race ./config/... ./dbsql/... ./orm/... ./observability/...

# 覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# e2e（需先启动 MySQL / Postgres / Redis / Kafka / Jaeger 容器）
go test -tags e2e -count=1 -timeout 300s ./e2e/...
```

e2e 端点全部可用环境变量覆盖，端口映射变化时无需改测试代码：

```text
DBX_E2E_MYSQL_HOST / _PORT / _USER / _PASS / _DB
DBX_E2E_PG_HOST    / _PORT / _USER / _PASS / _DB
DBX_E2E_REDIS_ADDR / DBX_E2E_REDIS_PASS
DBX_E2E_KAFKA_BROKER
DBX_E2E_JAEGER_OTLP / DBX_E2E_JAEGER_QUERY
```

**当前状态**：`gofmt` 干净、`go vet` 通过、`go test -race` 全绿；`e2e` 通过 `-tags e2e` 编译。

---

## 13. 主要依赖

| 类别 | 依赖 | 版本 | 用途 |
| :--- | :--- | :--- | :--- |
| ORM | `gorm.io/gorm` | v1.25.12 | 主 ORM |
| 驱动 | `gorm.io/driver/{mysql,postgres,sqlite,sqlserver}` | v1.5.x | 数据库驱动 |
| 集群 | `gorm.io/plugin/dbresolver` | v1.5.3 | 读写分离 |
| 配置解析 | `gopkg.in/yaml.v3` / `github.com/BurntSushi/toml` | v3.0.1 / v1.6.0 | YAML / TOML |
| 配置注入 | `github.com/mitchellh/mapstructure` | v1.5.0 | provider map → struct |
| MQ 适配 | `github.com/gospacex/mqx` | v0.0.1 | kafka / redis 连接池 |
| 追踪 | `go.opentelemetry.io/otel` + `otel/sdk` + `otel/trace` | v1.44.0 | OTel SDK |
| OTLP | `otel/exporters/otlp/otlptrace/otlptracehttp` | v1.32.0 | jaeger 导出 |
| 本地兄弟 | `configx` / `logx` / `hubx` | replace | 未发布，走本地路径 |

---

## 14. 示例

| 示例 | 演示内容 | 外部依赖 |
| :--- | :--- | :--- |
| [`examples/mysql-crud`](examples/mysql-crud/) | 4 个子命令 × 4 个 dbx 入口 + 5 步 CRUD + 3 路 exporter | MySQL、（可选）Redis / Kafka / Jaeger |
| [`examples/sqlite-configx`](examples/sqlite-configx/) | `xmiddleware.sqlx` 全契约键 + configx 入口 | 无（内嵌 SQLite） |

```bash
cd examples/mysql-crud
go run . p  -config mysql.example.yaml   # yaml + 单 MySQL
go run . o                               # 硬编码 + 单 MySQL
go run . pc -config cluster.example.yaml # yaml + 集群
go run . oc                              # 硬编码 + 集群
```

```bash
cd examples/sqlite-configx
go run .
```

---

## 15. 许可证

MIT License — 详见 [LICENSE](LICENSE)。
