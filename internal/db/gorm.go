package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GormDB struct {
	DB *gorm.DB
}

func ConnectGorm(ctx context.Context, databaseURL string) (*GormDB, error) {
	gormDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm db: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get gorm sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctxPing); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping gorm db: %w", err)
	}

	return &GormDB{DB: gormDB}, nil
}

func (g *GormDB) Close() {
	if g == nil || g.DB == nil {
		return
	}

	sqlDB, err := g.DB.DB()
	if err != nil {
		return
	}

	_ = sqlDB.Close()
}
