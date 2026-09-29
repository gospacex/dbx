# dbx 功能矩阵

> 维护方式：手工维护，改动功能时同步更新本表。
> 图例：✅ 已实现并有测试 / 🧪 已实现并有 e2e（docker）验证 / 🔒 预留未实现 / ❌ 不支持

## 1. ORM 驱动

| ORM driver | 状态 | 说明 | 案例/测试 |
|------------|------|------|-----------|
| gorm | ✅ | 唯一已实现的 ORM；`orm.Open` / `orm.OpenNoTrace` | `orm/*_test.go`、examples/mysql-crud |
| xorm | 🔒 | 契约保留 `xmiddleware.sqlx.driver: xorm`，monorepo 无消费方（YAGNI） | — |
| sqlx | 🔒 | 契约保留 `xmiddleware.sqlx.driver: sqlx`，monorepo 无消费方（YAGNI） | — |

## 2. 数据库方言（经 GORM dialector）

| 方言 | dialector | 单实例 | 读写分离集群 | hubx provider | 案例/测试 |
|------|-----------|--------|--------------|---------------|-----------|
| mysql | mysql | ✅ | ✅ | ✅ `hubx/mysqlx` | `e2e` MySQL_CRUD、`dbresolver` 测试 |
| postgres | postgres | ✅ | ✅ | ✅ `hubx/postgresx` | `e2e` Postgres_CRUD |
| sqlite | sqlite | ✅ | n/a | ✅ `hubx/sqlitex` | `dbsql/configx_test.go`、jaeger e2e |
| tidb | mysql（wire 兼容） | ✅ | ✅ | ✅ `hubx/tidbx` | `hubx/tidbx` 测试 |
| mariadb | mysql（wire 兼容） | ✅ | ✅ | — | `config` 加载测试 |
| gaussdb | postgres（wire 兼容） | ✅ | ✅ | — | `config` 加载测试 |
| mssql | sqlserver | ✅ | ✅ | — | `config` 加载测试 |
| oracle | ❌ | ❌ | ❌ | — | `orm.Dialector` 返回明确错误；原 `hubx/oraclex` 占位 provider 已删除（Build 恒失败） |
| dm / kingbase | 由 provider 自带 | ❌ | ❌ | — | 占位 provider（`hubx/dmx`、`hubx/kingbasex`）已删除：Build 恒返回 "driver not yet wired"，`client` 类型从未被构造 |

## 2.1 hubx registry provider（`hubx/<driver>x.Provider.Build`）

| provider | 注册名 | 建连方式 | 状态 |
|----------|--------|----------|------|
| `hubx/mysqlx` | `dbx.mysql` | `orm.Open`（GORM 句柄 + `*sql.DB`） | ✅ |
| `hubx/postgresx` | `dbx.postgres` | `sql.Open("postgres", dsn)` | ✅ |
| `hubx/sqlitex` | `dbx.sqlite` | `sql.Open("sqlite3", dsn)` | ✅ |
| `hubx/tidbx` | `dbx.tidb` | `sql.Open("mysql", dsn)` | ✅ |

> 四个 provider 的 `Config` 字段：`dsn` / `max_open_conns` / `max_idle_conns` / `conn_max_lifetime`。
> `Build` 返回的 `client` 是未导出类型，仅经 `hubx.Client` 接口暴露 `HealthCheck` / `Close`；
> `mysqlx` 的 `DB()` / `GORM()` 访问器外部无法到达（无法断言到未导出类型）。

## 3. 初始化方式（企业级连接构建）

| 方式 | API | 状态 | 案例/测试 |
|------|-----|------|-----------|
| 结构体直传 | `dbsql.Open(cfg DBConfig)` | ✅ | `dbsql/dbsql_test.go` |
| 配置文件（yaml/json/toml） | `dbsql.OpenPath(path)` | ✅ | `config/loader_test.go` |
| 读写分离集群 | `dbsql.OpenCluster / OpenClusterPath` | ✅ | `dbsql/cluster_test.go`、`e2e` |
| **configx 配置中心** | `dbsql.OpenFromConfigx(cfg, name, logger)` | ✅ | `dbsql/configx_test.go`（含对 configx 真实契约文件的对齐测试）+ `e2e/jaeger_e2e_test.go`（configx 入口真实 Jaeger 全链路） |
| hubx 注册中心 | `hubx/<driver>x.Provider.Build` | ✅ | `hubx/*/provider_test.go` |

## 4. 连接池

| 能力 | 状态 | 说明 |
|------|------|------|
| `PoolConfig`（秒级 int，`<=0` 取默认值 50/10/1800s/600s） | ✅ | `dbsql.applyPool` 统一应用 |
| configx duration 型（`5m`/`3m`，0 = 不限制） | ✅ | `OpenFromConfigx` 直接映射 `database/sql` |
| MaxIdleConns 自动封顶 MaxOpenConns | ✅ | `config/pool.go` |

## 5. 日志（logx）

| 能力 | 状态 | 说明 | 案例/测试 |
|------|------|------|-----------|
| logx 版 GORM logger | ✅ | `orm.NewLogxLogger` / `orm.WithLogxLogger`，nil 时回退 `logx.GetGlobal()`（无 provider 安全 no-op） | `orm/gorm_logx_test.go` |
| 慢查询阈值 `slow_threshold` | ✅ | 超过阈值按 warn 输出 | 同上 |
| 日志级别 `log_level`（silent/error/warn/info） | ✅ | 默认 warn | 同上 |
| 执行语句入日志 | ✅ | 字段 `db.statement` + `db.elapsed` + `db.rows` | 同上 |
| 日志-调用链关联 | ✅ | 走 logx `*Context` 方法，statement ctx 含活动 span 时自动注入链路上下文 | 结构性保证（logx 实现） |

## 6. 调用链（OTel）

| 能力 | 状态 | 说明 | 案例/测试 |
|------|------|------|-----------|
| Exporter：jaeger（OTLP/HTTP） | 🧪 | 裸 `host:port` 自动 WithInsecure | `dbsql/tracer_test.go` 回归测试 |
| Exporter：kafka（经 mqx kafkax.POS） | 🧪 | Shutdown 时 Flush 防丢 span | `e2e` Kafka |
| Exporter：redis_stream（POS/POC） | 🧪 | single/cluster 两种拓扑 | `e2e` RedisStream |
| **SQL 语句入链** | 🧪 | span 属性 `db.query.text`（prepared SQL，占位符不内联参数） | `e2e/jaeger_e2e_test.go`：`dbsql.Open` 与 `dbsql.OpenFromConfigx` 两条入口各自的 docker Jaeger 全链路断言 |
| `enable_trace` 开关 | ✅ | true 时 `OpenFromConfigx` 会创建全局 TracerProvider（否则 span 全部落到 noop）；false 时不改动调用方 provider | `TestOpenFromConfigx_TraceReachesCollector` / `TestOpenFromConfigx_TraceOff_RegistersNoProvider` |
| 采样（parentbased / traceidratio / always_off） | ✅ | `dbsql.parentBasedSampler` | `dbsql/tracer_test.go` |

## 7. 配置对齐（dbx ↔ configx）

| configx 契约键（`xmiddleware.sqlx.*`） | dbx 读取 | 状态 |
|----------------------------------------|----------|------|
| `driver` | `SQLConfig.Driver`（仅 gorm 实现） | ✅ |
| `databases.<name>.dsn` | `SQLConfig.Dsn`（必填） | ✅ |
| `databases.<name>.dialect`（附加键，缺省从 DSN 推断） | `SQLConfig.Dialect` | ✅ |
| `max_open_conns` / `max_idle_conns` | 连接池 | ✅ |
| `conn_max_lifetime` / `conn_max_idle_time` | 连接池（duration） | ✅ |
| `slow_threshold` | 慢查询日志 | ✅ |
| `enable_log` / `log_level` | logx GORM logger | ✅ |
| `enable_trace` | OTel 回调开关 | ✅ |
| `tracing.*`（`enabled`/`service`/`exporter`/`endpoint`/`sampler` 等） | `SQLConfig.Tracing`，`OpenFromConfigx` 据此创建全局 TracerProvider | ✅ |
| `health_check.*` | 读入 `SQLConfig.HealthCheck`；实际检查用 `db.DB().PingContext`，dbx 不起后台 pinger | ✅（半） |
| `retry.*` | 🔒 预留未实现（重试属服务层语义，GORM 无逐语句重试） | 🔒 |

> 对齐证明：`dbsql/configx_test.go` 中的 `TestReadConfigFromConfigx_RealContractFile`
> 直接读取 `configx/configs/config.simple.yaml` 并逐字段断言——契约文件改动会让测试变红。
>
> `tracing.*` 优先于 `enable_trace`；未配置 `tracing` 子段时按
> `TracingConfig.Validate()` 的默认值（jaeger + `localhost:4318`）处理。

## 8. 案例索引

| 案例 | 覆盖矩阵项 | 外部依赖 |
|------|-----------|----------|
| `examples/mysql-crud` | §1 gorm、§2 mysql、§3 结构体/文件、§4 连接池、§5 logx、§6 jaeger | MySQL、（可选）Jaeger |
| `examples/sqlite-configx` | §3 configx、§5 logx、§6 enable_trace、§7 全量契约键 | 无（sqlite 内嵌；可选 Jaeger） |
| `e2e/jaeger_e2e_test.go` | §6 SQL 语句入链（`Open` 与 `OpenFromConfigx` 两条入口） | docker jaeger-local |
| `e2e/db_e2e_test.go` | §2 mysql/postgres、§3 集群 | docker MySQL/Postgres |

> e2e 端点全部可用 `DBX_E2E_*` 环境变量覆盖（见 `e2e/endpoints_test.go`），
> 默认值与 tutorial `docker-compose.yml` 一致；端口被改映射时无需改测试代码。
