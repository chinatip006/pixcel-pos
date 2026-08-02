# Pixel CardShop POS — Backend (Go)

## สถานะตอนนี้ (Step 1 + 2 + 3)
ทำงานได้จริงแล้ว:
- `POST /api/auth/login`
- `GET/POST /api/products`, `PUT/PATCH/DELETE /api/products/:id`
- `POST   /api/transactions` — บันทึกบิล + ตัดสต๊อก + สะสมแต้มสมาชิก ใน 1 DB transaction เดียว
- `GET    /api/transactions?date=YYYY-MM-DD` — ดูประวัติบิลรายวัน
- `POST   /api/transactions/:orderId/cancel` — ยกเลิกบิล + คืนสต๊อก (ใช้ FOR UPDATE ล็อกแถวกันชนกัน)
- `GET    /api/reports/daily?date=YYYY-MM-DD` — สรุปยอดขายรายวัน

ยังไม่ทำ: open-tabs (พักบิล), members (สมาชิก+แต้ม)

## วิธีรันในเครื่องตัวเอง

1. ติดตั้ง Go 1.22+ (https://go.dev/dl/)
2. สร้างโปรเจกต์ Supabase แล้วรัน SQL ใน `migrations/0001_init.sql` ผ่าน Supabase SQL editor
3. คัดลอกข้อมูลพนักงานเข้าไปที่ตาราง `staff` อย่างน้อย 1 แถวเพื่อทดสอบ login เช่น:
   ```sql
   INSERT INTO staff (emp_id, name, role) VALUES ('001', 'ทดสอบ', 'staff');
   ```
4. คัดลอก `.env.example` เป็น `.env` แล้วใส่ `DATABASE_URL` จาก Supabase จริง
5. ดึง dependency แล้วรัน:
   ```bash
   cd backend
   go mod tidy
   set -a && source .env && set +a
   go run ./cmd/api
   ```
6. ทดสอบ:
   ```bash
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"emp_id":"001"}'
   ```
   ควรได้ `{"success":true,"token":"...","emp_id":"001","name":"ทดสอบ","role":"staff"}`

7. เอา token จากขั้นตอนที่ 6 มาทดสอบ endpoint สินค้า:
   ```bash
   TOKEN="วาง token ที่ได้ตรงนี้"

   # เพิ่มสินค้า
   curl -X POST http://localhost:8080/api/products \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"barcode":"8850001","name":"การ์ดหายาก","category":"การ์ด","price":120,"image_url":""}'

   # ดึงรายการสินค้า
   curl http://localhost:8080/api/products -H "Authorization: Bearer $TOKEN"

   # บันทึกบิล (สมมติมีสินค้าชื่อ "การ์ดหายาก" อยู่แล้ว)
   curl -X POST http://localhost:8080/api/transactions \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"cart":[{"name":"การ์ดหายาก","price":120,"unitPrice":120,"qty":1,"category":"การ์ด"}],"total":120,"cash":120,"cash_received":120,"change_amount":0}'

   # ดูประวัติบิลวันนี้
   curl "http://localhost:8080/api/transactions" -H "Authorization: Bearer $TOKEN"

   # สรุปยอดวันนี้
   curl "http://localhost:8080/api/reports/daily" -H "Authorization: Bearer $TOKEN"
   ```

## ขั้นตอนถัดไป (ยังไม่ทำในรอบนี้)
- เพิ่ม handlers: open-tabs (3 endpoint), members (4 endpoint)
- เพิ่ม unit test ของ `next_order_id()` และ transaction แบบ atomic ตอนบันทึกบิล
- ตั้งค่า deploy จริงบน Render (`render.yaml` จะเพิ่มในขั้น deploy)
