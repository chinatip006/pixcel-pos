package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool เปิด connection pool ไปหา Supabase Postgres
// ใช้ pgxpool เพราะ backend เป็น long-running service บน Render (ไม่ใช่ serverless)
// จึงเปิด pool ครั้งเดียวตอน start แล้วใช้ซ้ำได้ตลอดอายุ process
func NewPool(databaseURL string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
