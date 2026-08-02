package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pixelcardshop/pos-backend/internal/models"
)

type claimsWithJWT struct {
	EmpID string `json:"emp_id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken ออก JWT ให้พนักงานหลัง login สำเร็จ
func GenerateToken(secret string, staff models.Staff, expiryHours int) (string, error) {
	claims := claimsWithJWT{
		EmpID: staff.EmpID,
		Name:  staff.Name,
		Role:  staff.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   staff.EmpID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// parseToken ตรวจลายเซ็นและถอด claims กลับมา
func parseToken(secret, tokenStr string) (*claimsWithJWT, error) {
	claims := &claimsWithJWT{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return claims, nil
}

// RequireAuth คือ Gin middleware เช็ก "Authorization: Bearer <token>" ทุก request
// endpoint ที่ครอบด้วย middleware นี้ (คือทุกอันยกเว้น /api/auth/login และ /api/admin/*)
// ต้องแนบ token ที่ได้จาก login มาด้วยเสมอ
func RequireAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "ไม่พบ token กรุณาเข้าสู่ระบบใหม่"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := parseToken(jwtSecret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "session หมดอายุ กรุณาเข้าสู่ระบบใหม่"})
			return
		}

		// ฝัง emp_id/name/role ไว้ใน context ให้ handler ถัดไปดึงไปใช้
		// (ห้าม handler อ่าน emp_id จาก request body ของ client โดยตรงอีกต่อไป)
		c.Set("emp_id", claims.EmpID)
		c.Set("emp_name", claims.Name)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireCronSecret คือ middleware แยกสำหรับ endpoint /api/admin/* ที่ Render Cron Job เรียก
// ไม่ใช้ JWT พนักงาน เพราะเป็น job อัตโนมัติไม่มี "คนล็อกอิน" อยู่เบื้องหลัง
func RequireCronSecret(cronSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Cron-Secret") != cronSecret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			return
		}
		c.Next()
	}
}
