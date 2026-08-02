package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelcardshop/pos-backend/internal/auth"
	"github.com/pixelcardshop/pos-backend/internal/config"
	"github.com/pixelcardshop/pos-backend/internal/models"
)

type AuthHandler struct {
	DB  *pgxpool.Pool
	Cfg config.Config
}

func NewAuthHandler(db *pgxpool.Pool, cfg config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg}
}

type loginRequest struct {
	EmpID string `json:"emp_id" binding:"required"`
}

// Login เทียบเท่าฟังก์ชันเดิม checkLogin(empId)
// ต่างจากเดิมตรงที่สำเร็จแล้วจะออก JWT กลับไปด้วย ไม่ใช่แค่ยืนยันว่าเจอในชีต
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "กรุณากรอกรหัสพนักงาน"})
		return
	}
	empID := strings.TrimSpace(req.EmpID)

	var staff models.Staff
	row := h.DB.QueryRow(context.Background(),
		`SELECT emp_id, name, role FROM staff WHERE emp_id = $1`, empID)
	err := row.Scan(&staff.EmpID, &staff.Name, &staff.Role)

    if err == pgx.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "❌ ไม่พบรหัสพนักงานนี้ กรุณาลองใหม่"})
		return
	}
    if err != nil {
		// เปลี่ยนบรรทัดนี้ชั่วคราว เพื่อให้มันพ่น Error ออกมาดู
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "DB Error: " + err.Error()})
		return
	}

    token, err := auth.GenerateToken(h.Cfg.JWTSecret, staff, h.Cfg.JWTExpiryHours)
	if err != nil {
		// เปลี่ยนบรรทัดนี้ด้วยเช่นกัน
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Token Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
		"emp_id":  staff.EmpID,
		"name":    staff.Name,
		"role":    staff.Role,
	})
}
