package models

// Product สอดคล้องกับตาราง products ใน Postgres
// Stock เป็น *int เพราะ NULL หมายถึง "ขายไม่จำกัด" (ต่างจาก 0 ที่แปลว่าของหมด)
type Product struct {
	ID       int64   `json:"id"`
	Barcode  string  `json:"barcode"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"image_url"`
	Status   string  `json:"status"`
	Stock    *int    `json:"stock"` // null = ไม่จำกัด
}
