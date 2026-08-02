package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool สร้าง Database Connection Pool ที่ปลอดภัยสำหรับ Supabase
func NewPool(databaseURL string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. แปลง URL เป็น Config Object ของ pgx
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	// 2. ปิดการทำ Statement Caching เพื่อแก้ปัญหา prepared statement บน Supabase
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// 3. กำหนดจำนวน Connection Pool
	config.MaxConns = 10
	config.MinConns = 1

	// 4. สร้าง Pool จาก Config ที่ตั้งค่าเสร็จแล้ว
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool failed: %w", err)
	}

	// 5. ทดสอบการเชื่อมต่อ (Ping)
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database failed: %w", err)
	}

	return pool, nil
}
