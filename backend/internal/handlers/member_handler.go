package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelcardshop/pos-backend/internal/models"
)

type MemberHandler struct {
	DB *pgxpool.Pool
}

func NewMemberHandler(db *pgxpool.Pool) *MemberHandler {
	return &MemberHandler{DB: db}
}

// ---------- GET /api/members/:phone ----------
// เทียบเท่าฟังก์ชันเดิม getMemberByPhone(phone)
func (h *MemberHandler) GetByPhone(c *gin.Context) {
	phone := strings.TrimSpace(c.Param("phone"))

	var m models.Member
	row := h.DB.QueryRow(context.Background(), `
		SELECT phone, name, points, lifetime_total, grade, COALESCE(last_claim_month,'')
		FROM members WHERE phone = $1`, phone)
	err := row.Scan(&m.Phone, &m.Name, &m.Points, &m.LifetimeTotal, &m.Grade, &m.LastClaimMonth)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "❌ ไม่พบประวัติสมาชิกของเบอร์นี้"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true, "phone": m.Phone, "name": m.Name, "points": m.Points,
		"lifetime": m.LifetimeTotal, "grade": m.Grade, "last_claim_month": m.LastClaimMonth,
	})
}

// ---------- POST /api/members ----------
// เทียบเท่าฟังก์ชันเดิม registerNewMember(phone, name)
type registerMemberRequest struct {
	Phone string `json:"phone" binding:"required"`
	Name  string `json:"name" binding:"required"`
}

func (h *MemberHandler) Register(c *gin.Context) {
	var req registerMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "กรุณากรอกเบอร์โทรและชื่อ"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	name := strings.TrimSpace(req.Name)

	tag, err := h.DB.Exec(context.Background(), `
		INSERT INTO members (phone, name, points, lifetime_total, grade, last6mo_total, last_active_date)
		VALUES ($1, $2, 0, 0, 0, 0, (now() AT TIME ZONE 'Asia/Bangkok')::date)
		ON CONFLICT (phone) DO NOTHING`, phone, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "❌ เบอร์โทรศัพท์นี้เป็นสมาชิกอยู่แล้วครับ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "✅ สมัครสมาชิกคุณ " + name + " เรียบร้อยแล้ว!"})
}

// ---------- POST /api/members/:phone/redeem ----------
// เทียบเท่าฟังก์ชันเดิม redeemPoints(phone, pointsToDeduct, rewardName)
type redeemRequest struct {
	Points     int    `json:"points" binding:"required"`
	RewardName string `json:"reward_name" binding:"required"`
}

func (h *MemberHandler) Redeem(c *gin.Context) {
	phone := strings.TrimSpace(c.Param("phone"))
	var req redeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ข้อมูลไม่ครบ"})
		return
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "เริ่ม transaction ไม่สำเร็จ"})
		return
	}
	defer tx.Rollback(ctx)

	var currentPoints int
	err = tx.QueryRow(ctx, `SELECT points FROM members WHERE phone = $1 FOR UPDATE`, phone).Scan(&currentPoints)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "ไม่พบข้อมูลสมาชิก"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if currentPoints < req.Points {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "แต้มไม่พอแลกรางวัลนี้ครับ"})
		return
	}

	newPoints := currentPoints - req.Points
	if _, err := tx.Exec(ctx, `UPDATE members SET points = $1 WHERE phone = $2`, newPoints, phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "ยืนยัน transaction ไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "แลก " + req.RewardName + " สำเร็จ! คงเหลือ " + strconv.Itoa(newPoints) + " แต้ม",
		"new_points": newPoints,
	})
}

// ---------- POST /api/members/:phone/claim-freebie ----------
// เทียบเท่าฟังก์ชันเดิม claimMonthlyFreebie(phone, currentMonthStr)
type claimFreebieRequest struct {
	Month string `json:"month" binding:"required"` // รูปแบบ "YYYY-MM"
}

func (h *MemberHandler) ClaimFreebie(c *gin.Context) {
	phone := strings.TrimSpace(c.Param("phone"))
	var req claimFreebieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ข้อมูลไม่ครบ"})
		return
	}

	tag, err := h.DB.Exec(context.Background(),
		`UPDATE members SET last_claim_month = $1 WHERE phone = $2`, req.Month, phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "ไม่พบข้อมูลสมาชิกเบอร์นี้ในระบบ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "บันทึกการใช้สิทธิ์ลงฐานข้อมูลเรียบร้อย!"})
}
