package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelcardshop/pos-backend/internal/models"
	"github.com/pixelcardshop/pos-backend/internal/storage"
)

type ProductHandler struct {
	DB      *pgxpool.Pool
	Storage *storage.Client
}

func NewProductHandler(db *pgxpool.Pool, storageClient *storage.Client) *ProductHandler {
	return &ProductHandler{DB: db, Storage: storageClient}
}

// ---------- GET /api/products ----------
// เทียบเท่าฟังก์ชันเดิม getProducts()
func (h *ProductHandler) List(c *gin.Context) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT id, barcode, name, category, price, image_url, status, stock
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "อ่านข้อมูลสินค้าผิดพลาด"})
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

	var newID int64
	err := h.DB.QueryRow(context.Background(), `
		INSERT INTO products (barcode, name, category, price, image_url, status)
		VALUES (NULLIF($1,''), $2, $3, $4, $5, 'เปิด')
		RETURNING id`,
		req.Barcode, req.Name, req.Category, req.Price, req.ImageURL).Scan(&newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "เพิ่มสินค้าใหม่เรียบร้อยแล้ว!", "id": newID})
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

var allowedImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

const maxImageSizeBytes = 5 << 20 // 5 MB

// ---------- POST /api/products/:id/image ----------
// ฟังก์ชันใหม่ที่ระบบเดิมไม่มี — เดิมกรอก URL รูปเอง ตอนนี้อัปโหลดไฟล์จริงได้เลย
// รับไฟล์แบบ multipart/form-data ฟิลด์ชื่อ "image"
func (h *ProductHandler) UploadImage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "id ไม่ถูกต้อง"})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "กรุณาเลือกไฟล์รูปภาพ"})
		return
	}
	defer file.Close()

	if header.Size > maxImageSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ไฟล์รูปใหญ่เกินไป (จำกัด 5 MB)"})
		return
	}

	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "รองรับเฉพาะไฟล์ JPG, PNG, WEBP เท่านั้น"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "อ่านไฟล์ไม่สำเร็จ"})
		return
	}

	// ตั้งชื่อไฟล์ตาม id สินค้า + เวลา กันแคชรูปเก่าค้างในเบราว์เซอร์หลังแก้รูปใหม่
	objectPath := fmt.Sprintf("product-%d-%d.%s", id, time.Now().Unix(), ext)

	publicURL, err := h.Storage.UploadImage(objectPath, data, contentType)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "อัปโหลดรูปไม่สำเร็จ: " + err.Error()})
		return
	}

	tag2, err := h.DB.Exec(context.Background(),
		`UPDATE products SET image_url = $1, updated_at = now() WHERE id = $2`, publicURL, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tag2.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "ไม่พบสินค้านี้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "image_url": publicURL})
}
