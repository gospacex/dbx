// sqlite-configx 演示 dbx 的 configx 契约入口：
//
//	logx.New + SetGlobal            —— 业务日志与 GORM SQL 日志共用同一 logx 实例
//	configx.ReadConfig              —— 读取 ./config.yaml 得到统一配置句柄
//	dbsql.ReadConfigFromConfigx     —— 解析 xmiddleware.sqlx.databases.<name> 段
//	dbsql.OpenFromConfigx           —— 按契约建连（含连接池 / 日志 / trace 开关）
//
// 建连后执行一套 AutoMigrate + Create + Get + List + Update + Delete 的 CRUD 流程。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/dbx/dbsql"
	"github.com/gospacex/logx"
	// 注册 zap 日志实现：logx 根包不依赖具体 provider，必须 blank import
	// 一个实现包，否则 logx.New 会返回 "no logger provider registered"。
	_ "github.com/gospacex/logx/zap"

	"gorm.io/gorm"
)

// User 是本示例的最小领域实体。表结构刻意保持简单，焦点放在 dbx 的
// configx 契约建连流程上。
type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Age       int       `gorm:"not null;default:0" json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 固定表名，避免依赖 GORM 的复数化规则。
func (User) TableName() string { return "users" }

func main() {
	ctx := context.Background()

	// 1. logx：创建日志实例并设为全局。DefaultConfig 输出到 stdout、
	// console 格式，不产生 logs/ 目录。业务日志（*Context 方法）与
	// GORM 的 SQL 日志（OpenFromConfigx 内部经 orm.WithLogxLogger 接入）
	// 汇入同一实例，格式统一。
	lx, err := logx.New(logx.DefaultConfig())
	if err != nil {
		fatal("logx.New", err)
	}
	defer func() { _ = lx.Close() }()
	logx.SetGlobal(lx)

	// 2. configx：读取 ./config.yaml。句柄用完必须 Close（本地 viper
	// 模式下是 no-op，但远程配置中心模式下会断开 watch 连接）。
	cfg, err := configx.ReadConfig(configx.ConfigOptions{ConfigFile: "./config.yaml"})
	if err != nil {
		fatal("configx.ReadConfig", err)
	}
	defer func() { _ = cfg.Close() }()

	// 3. 解析契约配置并打印：ReadConfigFromConfigx 只读取
	// xmiddleware.sqlx.databases.<name> 段，缺省键保持零值，永不失败。
	sc := dbsql.ReadConfigFromConfigx(cfg, "default")
	lx.InfoContext(ctx, "sqlx config resolved",
		logx.String("driver", sc.Driver),
		logx.String("dialect", sc.Dialect),
		logx.String("dsn", sc.Dsn),
		logx.String("slow_threshold", sc.SlowThreshold.String()),
	)

	// 4. 按契约建连。第三个参数传入 logx 实例（传 nil 则回退
	// logx.GetGlobal()）；enable_log=true 时 GORM 的每条 SQL 都会以
	// db.statement 字段输出，enable_trace=false 时走 orm.OpenNoTrace。
	db, err := dbsql.OpenFromConfigx(cfg, "default", lx)
	if err != nil {
		fatal("dbsql.OpenFromConfigx", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		fatal("db.DB", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := runCRUD(ctx, db, lx); err != nil {
		fatal("crud", err)
	}

	lx.InfoContext(ctx, "example finished")
	fmt.Println("[stdout] CRUD sequence completed successfully")
}

// runCRUD 执行一套完整的 CRUD 流程，结果同时写入 logx（结构化日志）与
// stdout（便于肉眼验证）。
func runCRUD(ctx context.Context, db *gorm.DB, lx logx.Logger) error {
	if err := db.WithContext(ctx).AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	lx.InfoContext(ctx, "users table migrated")

	// Create
	alice := &User{Name: "alice", Age: 30}
	if err := db.WithContext(ctx).Create(alice).Error; err != nil {
		return fmt.Errorf("create: %w", err)
	}
	lx.InfoContext(ctx, "user created", logx.Int("id", int(alice.ID)), logx.String("name", alice.Name))
	fmt.Printf("[stdout] created:  id=%d name=%s age=%d\n", alice.ID, alice.Name, alice.Age)

	// Get
	var got User
	if err := db.WithContext(ctx).First(&got, alice.ID).Error; err != nil {
		return fmt.Errorf("get: %w", err)
	}
	lx.InfoContext(ctx, "user loaded", logx.Int("id", int(got.ID)), logx.String("name", got.Name))
	fmt.Printf("[stdout] got:      id=%d name=%s age=%d\n", got.ID, got.Name, got.Age)

	// List
	var users []User
	if err := db.WithContext(ctx).Limit(10).Find(&users).Error; err != nil {
		return fmt.Errorf("list: %w", err)
	}
	lx.InfoContext(ctx, "users listed", logx.Int("count", len(users)))
	fmt.Printf("[stdout] listed:   %d users\n", len(users))

	// Update
	if err := db.WithContext(ctx).Model(&User{}).Where("id = ?", got.ID).
		Update("age", 31).Error; err != nil {
		return fmt.Errorf("update: %w", err)
	}
	lx.InfoContext(ctx, "user updated", logx.Int("id", int(got.ID)), logx.Int("age", 31))
	fmt.Printf("[stdout] updated:  id=%d age=%d\n", got.ID, 31)

	// Delete：删一个不存在的 id，演示 RowsAffected=0 的软失败路径
	res := db.WithContext(ctx).Delete(&User{}, got.ID+1000)
	if res.Error != nil {
		return fmt.Errorf("delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// 行不存在不是错误，用 warn 提示即可
		lx.WarnContext(ctx, "delete matched no row", logx.Int("id", int(got.ID+1000)))
	}
	// 再删真实存在的行
	if err := db.WithContext(ctx).Delete(&User{}, got.ID).Error; err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	lx.InfoContext(ctx, "user deleted", logx.Int("id", int(got.ID)))
	fmt.Printf("[stdout] deleted:  id=%d\n", got.ID)

	return nil
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "[FATAL] %s: %v\n", step, err)
	os.Exit(1)
}
