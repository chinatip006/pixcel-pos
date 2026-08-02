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

	import "strings"

// เติมเงื่อนไขเช็คเพื่อต่อ ? หรือ & เข้าไปต่อท้าย URL 
   connString := databaseURL
    if !strings.Contains(connString, "?") {
    connString += "?default_query_exec_mode=simple_protocol"
    } else {
    connString += "&default_query_exec_mode=simple_protocol"
    }

dbPool, err := pgxpool.New(context.Background(), connString)
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
