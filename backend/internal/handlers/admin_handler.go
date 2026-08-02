package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminHandler struct {
	DB *pgxpool.Pool
}

func NewAdminHandler(db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{DB: db}
}

// ---------- POST /api/admin/downgrade-members ----------
// เทียบเท่าฟังก์ชันเดิม checkAndDowngradeMembers() ที่ของเดิมรันเป็น time-based trigger บน Apps Script
// ที่นี่ตั้งเป็น Render Cron Job ยิงเข้ามา (ดู render.yaml) ป้องกันด้วย X-Cron-Secret แทน JWT พนักงาน
// เงื่อนไข: สมาชิกเกรด > 0 ที่ไม่มียอดซื้อใน 6 เดือนล่าสุด (last6mo_total < 500)
// และไม่มีความเคลื่อนไหวมาแล้วเกิน 6 เดือน (last_active_date) จะถูกลดเกรดลง 1 ขั้น
// ทำเป็น SQL UPDATE เดียวจบ (ของเดิมต้อง loop ทีละแถวใน Apps Script)
func (h *AdminHandler) DowngradeMembers(c *gin.Context) {
	tag, err := h.DB.Exec(context.Background(), `
		UPDATE members
		SET grade = grade - 1,
		    last6mo_total = 0,
		    last_active_date = (now() AT TIME ZONE 'Asia/Bangkok')::date
		WHERE grade > 0
		  AND last6mo_total < 500
		  AND last_active_date <= (now() AT TIME ZONE 'Asia/Bangkok')::date - INTERVAL '6 months'`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "downgraded_count": tag.RowsAffected()})
}
