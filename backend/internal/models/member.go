package models

// Member สอดคล้องกับตาราง members
type Member struct {
	Phone          string  `json:"phone"`
	Name           string  `json:"name"`
	Points         int     `json:"points"`
	LifetimeTotal  float64 `json:"lifetime"`
	Grade          int     `json:"grade"`
	LastClaimMonth string  `json:"last_claim_month"`
}

// OpenTab คือบิลที่พักไว้ (ยังไม่จ่ายเงิน) สอดคล้องกับตาราง open_tabs
type OpenTab struct {
	TabID     string     `json:"tab_id"`
	TabName   string     `json:"tab_name"`
	Cart      []CartItem `json:"cart"`
	Remark    string     `json:"remark"`
	CreatedAt string     `json:"created_at"`
}
