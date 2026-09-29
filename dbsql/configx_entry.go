package dbsql

import (
	"fmt"
	"os"

	"github.com/gospacex/configx"
	"gorm.io/gorm"
)

// ======================= ConfigX 集成 =======================
//
// 与其余 x 系列 SDK 保持同一套入口三件套：
//
//	ReadConfig        只读配置，不开连接（预检/兼容用）
//	ReadAndCreate     读配置 → 打开 default 实例，返回可用 *gorm.DB
//	MustReadAndCreate 同上，失败即退出进程
//
// 与既有的 ReadConfigFromConfigx / OpenFromConfigx 的分工：那两个把
// configx 句柄当入参，适合"调用方已经持有句柄"或"要打开非 default 实例"
// 的场景；本文件的三件套则是"调用方只有配置来源"时的标准入口。

// defaultInstance ReadAndCreate 打开的实例名，对应
// xmiddleware.sqlx.databases.default。
const defaultInstance = "default"

// ReadConfig 加载统一配置句柄（所有 x 系列 SDK 共享同一配置源）。
func ReadConfig(opts ...configx.ConfigOptions) error {
	_, err := configx.ReadConfig(opts...)
	return err
}

// ReadAndCreate 从 configx 读取配置并打开 default 实例。
//
// 需要打开其它实例（如读写分离的 replica）时，改用
// ReadConfigFromConfigx + OpenFromConfigx 显式指定实例名。
//
// 调用方 MUST 在进程退出前 defer ShutdownTracerProvider(ctx)：配置里
// 打开 tracing 时本函数会装配全局 TracerProvider，不冲刷会丢尾部 span。
func ReadAndCreate(opts ...configx.ConfigOptions) (*gorm.DB, error) {
	c, err := configx.ReadConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("dbsql: failed to read config: %w", err)
	}
	// 日志句柄传 nil：logx 适配器会回落到 logx.GetGlobal()，无 provider
	// 时是安全空实现，不改变调用方可观测性配置。
	return OpenFromConfigx(c, defaultInstance, nil)
}

// MustReadAndCreate 类似 ReadAndCreate，失败时退出进程
func MustReadAndCreate(opts ...configx.ConfigOptions) *gorm.DB {
	db, err := ReadAndCreate(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbsql: failed to open database: %v\n", err)
		os.Exit(1)
	}
	return db
}
