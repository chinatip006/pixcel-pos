package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelcardshop/pos-backend/internal/models"
)

type OpenTabHandler struct {
	DB *pgxpool.Pool
}

func NewOpenTabHandler(db *pgxpool.Pool) *OpenTabHandler {
	return &OpenTabHandler{DB: db}
}

// ---------- POST /api/open-tabs ----------
// เทียบเท่าฟังก์ชันเดิม saveOpenTab(tabName, cartStr, remark)
type saveOpenTabRequest struct {
	TabName string            `json:"tab_name" binding:"required"`
	Cart    []models.CartItem `json:"cart" binding:"required"`
	Remark  string            `json:"remark"`
}

func (h *OpenTabHandler) Create(c *gin.Context) {
	var req saveOpenTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ข้อมูลไม่ครบ"})
		return
	}

	cartJSON, err := json.Marshal(req.Cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "แปลงข้อมูลตะกร้าผิดพลาด"})
		return
	}

	// tab_id รูปแบบเดียวกับของเดิม "TAB" + timestamp มิลลิวินาที
	tabID := fmt.Sprintf("TAB%d", time.Now().UnixMilli())

	_, err = h.DB.Exec(context.Background(), `
		INSERT INTO open_tabs (tab_id, tab_name, cart, remark)
		VALUES ($1, $2, $3::jsonb, $4)`, tabID, req.TabName, string(cartJSON), req.Remark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "tab_id": tabID})
}

// ---------- GET /api/open-tabs ----------
// เทียบเท่าฟังก์ชันเดิม getOpenTabs()
func (h *OpenTabHandler) List(c *gin.Context) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT tab_id, tab_name, cart, remark, to_char(created_at AT TIME ZONE 'Asia/Bangkok', 'DD/MM/YYYY HH24:MI:SS')
		FROM open_tabs ORDER BY created_at`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "โหลดบิลที่พักไว้ไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	tabs := []models.OpenTab{}
	for rows.Next() {
		var t models.OpenTab
		var cartRaw []byte
		if err := rows.Scan(&t.TabID, &t.TabName, &cartRaw, &t.Remark, &t.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "อ่านข้อมูลผิดพลาด"})
			return
		}
		_ = json.Unmarshal(cartRaw, &t.Cart)
		tabs = append(tabs, t)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": tabs})
}

// ---------- DELETE /api/open-tabs/:tabId ----------
// เทียบเท่าฟังก์ชันเดิม deleteOpenTab(rowNumber) — ใช้ tab_id แทนเลขแถว
func (h *OpenTabHandler) Delete(c *gin.Context) {
	tabID := c.Param("tabId")

	tag, err := h.DB.Exec(context.Background(), `DELETE FROM open_tabs WHERE tab_id = $1`, tabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "ไม่พบบิลพักนี้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
