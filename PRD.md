# SDK 清单:
一款支持 8 大数据库的 GORM 接入工具包：基于缓存单例机制，只需 P(path) 或 O(obj) 一行代码，即可快速获取生产级 *gorm.DB 句柄，并自动完成单机/集群、连接池调优及读写分离等复杂配置。
输入输出： P(path) / O(obj) 一行代码获取 *gorm.DB
能力边界： 8种数据库、单机/集群双模式、连接池、读写分离
底层机制： 缓存单例机制、生产级自动配置

## mod:
github.com/gospacex/dbx

## GORM 官方技术栈清单

### 一、核心 ORM 库

| 组件 | 导入路径 | 说明 |
| :--- | :--- | :--- |
| **GORM 核心** | `gorm.io/gorm` | 主框架，提供所有 ORM 基础功能 |

### 二、官方数据库驱动

| 数据库 | 驱动包导入路径 | 备注 |
| :--- | :--- | :--- |
| **MySQL** | `gorm.io/driver/mysql` | 最常用的驱动之一 |
| **PostgreSQL** | `gorm.io/driver/postgres` | 功能完善，支持 JSONB 等特性 |
| **SQLite** | `gorm.io/driver/sqlite` | 轻量级嵌入式数据库 |
| **SQL Server** | `gorm.io/driver/sqlserver` | 微软 SQL Server 数据库 |
| **Oracle** | `gorm.io/driver/oracle` | 支持 Oracle 数据库 |
| **GaussDB** | `gorm.io/driver/gaussdb` | 华为高斯数据库 |
| **TiDB** | `gorm.io/driver/mysql` | 使用 MySQL 驱动即可连接 |
| **MariaDB** | `gorm.io/driver/mysql` | 兼容 MySQL 驱动 |

### 三、集群扩展插件

| 插件 | 导入路径 | 安装命令 | 功能说明 |
| :--- | :--- | :--- | :--- |
| **DBResolver** | `gorm.io/plugin/dbresolver` | `go get gorm.io/plugin/dbresolver` | 多数据库支持、读写分离、负载均衡、集群连接 |

### 💡 注意事项

1. **V1 vs V2**：以上所有驱动均为 **GORM V2** 官方版本，旧版 `github.com/jinzhu/gorm` 已停止维护
2. **TiDB/MariaDB**：均复用 MySQL 驱动，使用 `gorm.io/driver/mysql` 即可
3. **ClickHouse 等数据库**：需使用第三方驱动，GORM 官方暂未提供

如果需要特定数据库的连接配置示例或集群方案的详细代码，可以告诉我！

两种入参格式：
1.object，结构体对象
2.path，yaml配置路径

两种对接环境: 单机、集群

两种快捷方法：
1.P(path), 返回各种gorm对接的中间件句柄
2.O(object),返回各种gorm对接的中间件句柄

实例化思路：
用 sync.Map 按配置 key 缓存，用 sync.Once 保证每个 key 只初始化一次实现快捷函数

配置路径读取
梳理各种对接数据库的单机和集群生产级别结构体入参

维度	建议
连接池调优	MaxOpenConns 建议 ≤ 数据库最大连接数 / 实例数。云数据库通常建议 50~200，避免频繁建连与 OOM。
事务路由	DBResolver 默认在事务内强制走 Sources。若需强制读从库，需使用 db.Clauses(dbresolver.Read).Find()。
副本延迟	读写分离场景必须监控 Replica Lag。关键业务（如支付后查订单）应强制走主库或加读前延迟补偿。
TLS 证书	生产环境务必启用 verify-full。证书热更新需配合 crypto/tls.Config.GetCertificate 动态加载，避免重启。
优雅关闭	应用退出前调用 sqlDB, _ := db.DB(); sqlDB.Close()，配合 context.WithTimeout 等待活跃事务结束。
指标暴露	建议集成 prometheus 中间件：gorm.io/plugin/prometheus，暴露 gorm_pool_open, gorm_slow_query 等指标。
