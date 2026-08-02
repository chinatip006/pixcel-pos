package config

import (
	"log"
	"os"
)

// Config รวม env vars ทั้งหมดที่ backend ต้องใช้
type Config struct {
	Port           string // ค่า default Render จะฉีดผ่าน env PORT มาให้อัตโนมัติ
	DatabaseURL    string // Supabase connection string (session pooler)
	JWTSecret      string // ใช้เซ็น/ตรวจ JWT ตอน login
	CronSecret     string // ใช้ป้องกัน endpoint /api/admin/* ที่ยิงจาก Render Cron Job
	AllowedOrigin  string // โดเมน Firebase Hosting ของ frontend สำหรับตั้งค่า CORS
	JWTExpiryHours int    // อายุ token กี่ชั่วโมง (ปรับตามรอบกะพนักงาน)
}

// Load อ่านค่าจาก environment variable ทั้งหมด แล้ว fail-fast ถ้าค่าที่จำเป็นหายไป
// (จำเป็น เพราะ deploy บน Render ต้องตั้ง env vars ให้ครบ ไม่งั้นเซอร์วิสไม่ควรจะรันขึ้นมาเงียบๆ)
func Load() Config {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    mustEnv("DATABASE_URL"),
		JWTSecret:      mustEnv("JWT_SECRET"),
		CronSecret:     mustEnv("CRON_SECRET"),
		AllowedOrigin:  mustEnv("ALLOWED_ORIGIN"),
		JWTExpiryHours: 8,
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}
