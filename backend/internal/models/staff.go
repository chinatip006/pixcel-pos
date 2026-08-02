package models

// Staff สอดคล้องกับตาราง staff ใน Postgres
type Staff struct {
	EmpID string `json:"emp_id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// Claims คือข้อมูลที่ฝังลงใน JWT ตอน login สำเร็จ
// handler อื่นๆ จะอ่าน emp_id/role จากตรงนี้แทนการเชื่อ client ที่ส่งมาเอง
// (จุดนี้คือช่องโหว่ของระบบเดิมที่รับ empInfo ตรงๆ จาก frontend)
type Claims struct {
	EmpID string `json:"emp_id"`
	Role  string `json:"role"`
}
