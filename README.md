# Pixel CardShop POS — ย้ายระบบจาก Google Apps Script

ระบบเดิม (Apps Script + Google Sheets) ย้ายมาเป็น:
- **Frontend:** Firebase Hosting (`frontend/`)
- **Backend:** Go REST API บน Render (`backend/`)
- **Database:** Supabase (PostgreSQL)
- **Source control:** GitHub

โค้ดครบทุกฟังก์ชันของระบบเดิมแล้ว (login, สินค้า, ขาย/พักบิล/ยกเลิกบิล, ระบบสมาชิก+แต้ม+เกรด, แดชบอร์ดยอดขาย)

---

## ขั้นตอน Deploy แบบ end-to-end

### 1. Supabase (ฐานข้อมูล)
1. สร้างโปรเจกต์ใหม่ที่ [supabase.com](https://supabase.com)
2. เปิด SQL Editor → รันไฟล์ `backend/migrations/0001_init.sql` ทั้งไฟล์
3. เพิ่มพนักงานอย่างน้อย 1 คนไว้ทดสอบ:
   ```sql
   INSERT INTO staff (emp_id, name, role) VALUES ('001', 'ชื่อพนักงาน', 'staff');
   ```
4. ไปที่ Project Settings → Database → คัดลอก **Connection string** โหมด **Session pooler** เก็บไว้ (จะใช้เป็น `DATABASE_URL`)

### 2. GitHub
1. สร้าง repo ใหม่ (private ก็ได้)
2. จากในโฟลเดอร์ที่แตก zip นี้:
   ```bash
   git init
   git add .
   git commit -m "Initial migration from Google Apps Script"
   git branch -M main
   git remote add origin https://github.com/<your-username>/<your-repo>.git
   git push -u origin main
   ```

### 3. Render (backend)
**วิธีที่ง่ายที่สุด — ใช้ `render.yaml`:**
1. ไปที่ Render Dashboard → New → Blueprint → เลือก repo ที่เพิ่ง push
2. Render จะอ่าน `render.yaml` แล้วสร้าง Web Service + Cron Job ให้อัตโนมัติ
3. เข้าไปตั้งค่า env vars ที่ยังไม่ auto-generate:
   - `DATABASE_URL` = connection string จาก Supabase (ขั้นตอนที่ 1.4)
   - `ALLOWED_ORIGIN` = โดเมน Firebase Hosting (จะได้หลัง deploy frontend ขั้นตอนที่ 4 — ใส่ทีหลังได้)
   - Cron job ต้องใส่ `CRON_SECRET` ให้ตรงกับของ Web Service
4. รอ deploy เสร็จ จะได้ URL backend เช่น `https://pixel-pos-api.onrender.com`
5. ทดสอบ: `curl https://pixel-pos-api.onrender.com/healthz` ควรได้ `{"status":"ok"}`

### 4. Firebase Hosting (frontend)
1. แก้ `frontend/config.js` ให้ `API_BASE_URL` ตรงกับ URL backend จริงจากขั้นตอนที่ 3
2. ติดตั้ง Firebase CLI: `npm install -g firebase-tools`
3. ```bash
   cd frontend
   firebase login
   firebase init hosting   # เลือกโฟลเดอร์ปัจจุบันเป็น public dir, single-page app = No
   firebase deploy
   ```
4. จะได้โดเมนเช่น `https://your-project.web.app` — เอาไปใส่ใน `ALLOWED_ORIGIN` ของ Render (ขั้นตอนที่ 3.3) แล้ว restart service

### 5. ทดสอบ end-to-end
1. เปิดโดเมน Firebase Hosting
2. ลอง login ด้วยรหัสพนักงานที่ insert ไว้ในขั้นตอนที่ 1.3
3. ลองเพิ่มสินค้า → ขาย → ดูประวัติบิล → ยกเลิกบิล → สมัครสมาชิก → แลกแต้ม ให้ครบทุกฟังก์ชัน

---

## โครงสร้าง repo

```
pixel-pos/
├── backend/               # Go REST API
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── auth/          # JWT ออก/ตรวจ token
│   │   ├── config/        # อ่าน env vars
│   │   ├── db/             # Postgres connection pool
│   │   ├── handlers/       # HTTP handlers (auth, products, transactions, open-tabs, members, admin)
│   │   └── models/
│   ├── migrations/0001_init.sql
│   ├── go.mod
│   ├── .env.example
│   └── README.md           # วิธีรัน backend ในเครื่อง + ทดสอบทีละ endpoint
├── frontend/               # เว็บหน้าร้าน (ไฟล์เดิมปรับแล้ว)
│   ├── index.html
│   ├── api.js               # ตัวแทน google.script.run เดิม เปลี่ยนไปยิง REST API จริง
│   ├── config.js            # ตั้ง URL backend ที่นี่
│   └── firebase.json
├── render.yaml              # deploy backend + cron job อัตโนมัติ
└── .gitignore
```

## สิ่งที่ปรับปรุงจากระบบเดิม
- **Auth จริง:** เดิมกรอกรหัสพนักงานผ่านแล้วเรียก endpoint ไหนก็ได้ ตอนนี้ต้องมี JWT ที่ได้จาก login เท่านั้น
- **emp_info ปลอมไม่ได้:** ระบบเดิมรับชื่อพนักงานจาก frontend ตรงๆ ตอนนี้ backend ดึงจาก JWT เท่านั้น
- **แก้ไข/ลบ/ยกเลิกสินค้าด้วย id:** เดิมใช้ชื่อ ซึ่งพังถ้าชื่อซ้ำหรือมีช่องว่างเกิน
- **บันทึกบิลเป็น atomic transaction:** บันทึกบิล + ตัดสต๊อก + สะสมแต้ม อยู่ใน DB transaction เดียว ถ้าพลาดขั้นไหน rollback หมด
- **เลขบิลไม่ชนกัน:** ใช้ Postgres function ออกเลขบิลแบบ atomic รองรับขายพร้อมกันหลายเครื่อง
- **ยกเลิกบิลกันชนกัน:** ใช้ `SELECT ... FOR UPDATE` ล็อกแถวระหว่างยกเลิก
