package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelcardshop/pos-backend/internal/models"
)

type TransactionHandler struct {
	DB *pgxpool.Pool
}

func NewTransactionHandler(db *pgxpool.Pool) *TransactionHandler {
	return &TransactionHandler{DB: db}
}

// ---------- POST /api/transactions ----------
// เทียบเท่าฟังก์ชันเดิม saveTransaction(...) ที่ทำ 3 อย่างพร้อมกัน:
// 1) บันทึกบิล 2) ตัดสต๊อก 3) สะสมแต้ม/อัปเกรดเกรดสมาชิก
// ของเดิมทำทีละสเต็ปบน Google Sheets (เสี่ยงพังครึ่งทาง) ที่นี่ทำเป็น "1 DB transaction เดียว"
// ถ้าขั้นไหนพลาดจะ rollback ทั้งหมด ไม่มีทางที่บิลถูกบันทึกแต่สต๊อกไม่ถูกตัด
type createTransactionRequest struct {
	Cart         []models.CartItem `json:"cart" binding:"required"`
	Total        float64           `json:"total" binding:"required"`
	Cash         float64           `json:"cash"`
	CashReceived float64           `json:"cash_received"`
	ChangeAmount float64           `json:"change_amount"`
	Transfer     float64           `json:"transfer"`
	Remark       string            `json:"remark"`
	MemberPhone  string            `json:"member_phone"`
}

func (h *TransactionHandler) Create(c *gin.Context) {
	var req createTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Cart) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ตะกร้าว่าง หรือข้อมูลไม่ครบ"})
		return
	}

	// emp_info มาจาก JWT เท่านั้น ไม่รับจาก client โดยตรง (ของเดิมรับ empInfo ตรงๆ จาก frontend ซึ่งปลอมได้)
	empID := c.GetString("emp_id")
	empName := c.GetString("emp_name")
	empInfo := empID + " - " + empName

	cartJSON, err := json.Marshal(req.Cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "แปลงข้อมูลตะกร้าผิดพลาด"})
		return
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "เริ่ม transaction ไม่สำเร็จ"})
		return
	}
	defer tx.Rollback(ctx) // no-op ถ้า commit สำเร็จไปแล้ว

	// 1) ออกเลขบิลแบบ atomic
	var orderID string
	if err := tx.QueryRow(ctx, `SELECT next_order_id()`).Scan(&orderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "ออกเลขบิลไม่สำเร็จ"})
		return
	}

	// 2) บันทึกบิล
	var memberPhone interface{}
	if req.MemberPhone != "" {
		memberPhone = req.MemberPhone
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO transactions
			(order_id, cart, total, cash, cash_received, change_amount, transfer, emp_info, remark, member_phone, status)
		VALUES ($1, $2::jsonb, $3, $4, $5, $6, $7, $8, $9, $10, 'ปกติ')`,
		orderID, string(cartJSON), req.Total, req.Cash, req.CashReceived, req.ChangeAmount,
		req.Transfer, empInfo, req.Remark, memberPhone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "บันทึกบิลไม่สำเร็จ: " + err.Error()})
		return
	}

	// 3) ตัดสต๊อกทีละรายการ (เฉพาะสินค้าที่ตั้ง stock ไว้ ไม่ใช่ NULL ที่แปลว่าขายไม่จำกัด)
	// กันสต๊อกติดลบด้วย GREATEST(...,0) เหมือนของเดิม
	for _, item := range req.Cart {
		_, err = tx.Exec(ctx, `
			UPDATE products SET stock = GREATEST(stock - $1, 0), updated_at = now()
			WHERE name = $2 AND stock IS NOT NULL`, item.Qty, item.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "ตัดสต๊อกไม่สำเร็จ: " + err.Error()})
			return
		}
	}

	// 4) สะสมแต้ม/อัปเกรดเกรดสมาชิก (ถ้ามีการผูกเบอร์สมาชิกกับบิลนี้)
	if req.MemberPhone != "" {
		_, err = tx.Exec(ctx, `
			UPDATE members SET
				points = points + FLOOR($1::numeric / 50),
				lifetime_total = lifetime_total + $1,
				last6mo_total = last6mo_total + $1,
				last_active_date = (now() AT TIME ZONE 'Asia/Bangkok')::date,
				grade = CASE
					WHEN lifetime_total + $1 >= 20000 THEN 4
					WHEN lifetime_total + $1 >= 10000 THEN 3
					WHEN lifetime_total + $1 >= 5000  THEN 2
					WHEN lifetime_total + $1 >= 1500  THEN 1
					ELSE 0
				END
			WHERE phone = $2`, req.Total, req.MemberPhone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "อัปเดตแต้มสมาชิกไม่สำเร็จ: " + err.Error()})
			return
		}
		// หมายเหตุ: ถ้าไม่เจอเบอร์นี้ในตาราง members จะไม่มี error (UPDATE ไม่เจอแถวก็แค่ไม่ทำอะไร)
		// ต่างจากของเดิมที่ silent fail เหมือนกัน จึงคงพฤติกรรมเดิมไว้
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "ยืนยัน transaction ไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "order_id": orderID})
}

// ---------- GET /api/transactions?date=YYYY-MM-DD ----------
// เทียบเท่าฟังก์ชันเดิม getTransactionHistory(targetDate)
func (h *TransactionHandler) History(c *gin.Context) {
	date := c.Query("date") // ว่างได้ ถ้าว่างจะใช้วันนี้ (เวลาไทย)

	rows, err := h.DB.Query(context.Background(), `
		SELECT order_id, to_char(created_at AT TIME ZONE 'Asia/Bangkok', 'HH24:MI:SS'),
		       cart, total, emp_info, status
		FROM transactions
		WHERE (created_at AT TIME ZONE 'Asia/Bangkok')::date =
		      COALESCE(NULLIF($1,'')::date, (now() AT TIME ZONE 'Asia/Bangkok')::date)
		ORDER BY created_at DESC`, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "โหลดประวัติบิลไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	history := []models.TransactionSummary{}
	for rows.Next() {
		var t models.TransactionSummary
		var cartRaw []byte
		if err := rows.Scan(&t.OrderID, &t.Time, &cartRaw, &t.Total, &t.EmpInfo, &t.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "อ่านข้อมูลบิลผิดพลาด"})
			return
		}
		_ = json.Unmarshal(cartRaw, &t.Cart)
		history = append(history, t)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": history})
}

// ---------- POST /api/transactions/:orderId/cancel ----------
// เทียบเท่าฟังก์ชันเดิม cancelBill(orderId, empId)
// ต่างจากเดิม: ไม่ต้องรับ empId มาใน body แล้วเช็คซ้ำ เพราะ JWT ยืนยันตัวตนอยู่แล้ว
func (h *TransactionHandler) Cancel(c *gin.Context) {
	orderID := c.Param("orderId")
	empName := c.GetString("emp_name")

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "เริ่ม transaction ไม่สำเร็จ"})
		return
	}
	defer tx.Rollback(ctx)

	var status string
	var cartRaw []byte
	err = tx.QueryRow(ctx, `SELECT status, cart FROM transactions WHERE order_id = $1 FOR UPDATE`, orderID).
		Scan(&status, &cartRaw)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "ไม่พบบิลเลขที่ " + orderID})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if status == "ยกเลิก" {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "บิลนี้ถูกยกเลิกไปแล้ว"})
		return
	}

	_, err = tx.Exec(ctx, `
		UPDATE transactions SET status = 'ยกเลิก', cancelled_by = $1, cancelled_at = now()
		WHERE order_id = $2`, empName, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "ยกเลิกบิลไม่สำเร็จ"})
		return
	}

	var cart []models.CartItem
	_ = json.Unmarshal(cartRaw, &cart)
	for _, item := range cart {
		_, err = tx.Exec(ctx, `
			UPDATE products SET stock = stock + $1, updated_at = now()
			WHERE name = $2 AND stock IS NOT NULL`, item.Qty, item.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "คืนสต๊อกไม่สำเร็จ"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "ยืนยัน transaction ไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "ยกเลิกบิลเลขที่ " + orderID + " และคืนสต๊อกเรียบร้อยแล้ว!"})
}

// ---------- GET /api/reports/daily?date=YYYY-MM-DD ----------
// เทียบเท่าฟังก์ชันเดิม getDailySummary(targetDate)
func (h *TransactionHandler) DailySummary(c *gin.Context) {
	date := c.Query("date")

	rows, err := h.DB.Query(context.Background(), `
		SELECT cart, total, cash, transfer
		FROM transactions
		WHERE (created_at AT TIME ZONE 'Asia/Bangkok')::date =
		      COALESCE(NULLIF($1,'')::date, (now() AT TIME ZONE 'Asia/Bangkok')::date)
		  AND status != 'ยกเลิก'`, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "สรุปยอดขายไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	summary := models.DailySummary{}
	itemMap := map[string]*models.ItemSaleSummary{}

	for rows.Next() {
		var cartRaw []byte
		var total, cash, transfer float64
		if err := rows.Scan(&cartRaw, &total, &cash, &transfer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "อ่านข้อมูลผิดพลาด"})
			return
		}
		summary.Bills++
		summary.Total += total
		summary.Cash += cash
		summary.Transfer += transfer

		var cart []models.CartItem
		_ = json.Unmarshal(cartRaw, &cart)
		for _, item := range cart {
			entry, ok := itemMap[item.Name]
			if !ok {
				entry = &models.ItemSaleSummary{Name: item.Name}
				itemMap[item.Name] = entry
			}
			entry.Qty += item.Qty
			entry.Total += item.Price
		}
	}

	for _, v := range itemMap {
		summary.Items = append(summary.Items, *v)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}
