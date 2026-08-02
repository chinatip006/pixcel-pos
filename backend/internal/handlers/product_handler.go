package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelcardshop/pos-backend/internal/models"
)

type ProductHandler struct {
	DB *pgxpool.Pool
}

func NewProductHandler(db *pgxpool.Pool) *ProductHandler {
	return &ProductHandler{DB: db}
}

// ---------- GET /api/products ----------
// เทียบเท่าฟังก์ชันเดิม getProducts()
func (h *ProductHandler) List(c *gin.Context) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT 
			id, 
			COALESCE(barcode, '') AS barcode, 
			name, 
			COALESCE(category, '') AS category, 
			price, 
			COALESCE(image_url, '') AS image_url, 
			status, 
			stock
		FROM products
		ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "โหลดสินค้าไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Barcode, &p.Name, &p.Category, &p.Price, &p.ImageURL, &p.Status, &p.Stock); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Scan Error: " + err.Error()})
			return
		}
		products = append(products, p)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": products})
}
// ---------- POST /api/products ----------
// เทียบเท่าฟังก์ชันเดิม addNewProduct(barcode, name, category, price, imageUrl)
type addProductRequest struct {
	Barcode  string  `json:"barcode"`
	Name     string  `json:"name" binding:"required"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"image_url"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req addProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "กรุณากรอกชื่อสินค้า"})
		return
	}

	_, err := h.DB.Exec(context.Background(), `
		INSERT INTO products (barcode, name, category, price, image_url, status)
		VALUES (NULLIF($1,''), $2, $3, $4, $5, 'เปิด')`,
		req.Barcode, req.Name, req.Category, req.Price, req.ImageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "เพิ่มสินค้าใหม่เรียบร้อยแล้ว!"})
}

// ---------- PUT /api/products/:id ----------
// เทียบเท่าฟังก์ชันเดิม updateProductDetails(oldName, newData)
// ต่างจากเดิม: ใช้ id แทนการค้นด้วยชื่อ (กันบั๊กชื่อซ้ำ/มีช่องว่างเกิน)
type updateProductRequest struct {
	Barcode  string  `json:"barcode"`
	Name     string  `json:"name" binding:"required"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"image_url"`
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "id ไม่ถูกต้อง"})
		return
	}

	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ข้อมูลไม่ครบ"})
		return
	}

	tag, err := h.DB.Exec(context.Background(), `
		UPDATE products
		SET barcode = NULLIF($1,''), name = $2, category = $3, price = $4, image_url = $5, updated_at = now()
		WHERE id = $6`,
		req.Barcode, req.Name, req.Category, req.Price, req.ImageURL, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "❌ ไม่พบสินค้านี้ในระบบ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "อัปเดตข้อมูลสำเร็จ"})
}

// ---------- PATCH /api/products/:id/status ----------
// เทียบเท่าฟังก์ชันเดิม toggleProductStatus(name, currentStatus)
func (h *ProductHandler) ToggleStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "id ไม่ถูกต้อง"})
		return
	}

	tag, err := h.DB.Exec(context.Background(), `
		UPDATE products
		SET status = CASE WHEN status = 'เปิด' THEN 'ปิด' ELSE 'เปิด' END,
		    updated_at = now()
		WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "❌ ไม่พบสินค้านี้ อาจถูกลบไปแล้ว"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------- PATCH /api/products/:id/stock ----------
// เทียบเท่าฟังก์ชันเดิม updateProductStock(name, newStock)
// ส่ง "stock": null มาเพื่อเซ็ตเป็นขายไม่จำกัด
type updateStockRequest struct {
	Stock *int `json:"stock"`
}

func (h *ProductHandler) UpdateStock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "id ไม่ถูกต้อง"})
		return
	}

	var req updateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	tag, err := h.DB.Exec(context.Background(),
		`UPDATE products SET stock = $1, updated_at = now() WHERE id = $2`, req.Stock, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "❌ ไม่พบสินค้านี้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------- DELETE /api/products/:id ----------
// เทียบเท่าฟังก์ชันเดิม deleteProduct(name)
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "id ไม่ถูกต้อง"})
		return
	}

	tag, err := h.DB.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "❌ ไม่พบสินค้านี้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
