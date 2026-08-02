package models

// CartItem คือ 1 รายการสินค้าในตะกร้า/บิล เก็บเป็น jsonb ในตาราง transactions
type CartItem struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	UnitPrice float64 `json:"unitPrice"`
	Qty      int     `json:"qty"`
	Category string  `json:"category"`
}

// TransactionSummary ใช้ตอบกลับตอนดูประวัติบิลรายวัน
type TransactionSummary struct {
	OrderID string     `json:"order_id"`
	Time    string     `json:"time"`
	Cart    []CartItem `json:"cart"`
	Total   float64    `json:"total"`
	EmpInfo string     `json:"emp_info"`
	Status  string     `json:"status"`
}

// DailySummary คือผลสรุปยอดขายรายวัน เทียบเท่าฟังก์ชันเดิม getDailySummary()
type DailySummary struct {
	Bills    int              `json:"bills"`
	Total    float64          `json:"total"`
	Cash     float64          `json:"cash"`
	Transfer float64          `json:"transfer"`
	Items    []ItemSaleSummary `json:"items"`
}

type ItemSaleSummary struct {
	Name  string  `json:"name"`
	Qty   int     `json:"qty"`
	Total float64 `json:"total"`
}
