-- รันไฟล์นี้ก่อน แล้วค่อยรัน migrations/0001_init.sql ใหม่อีกรอบ
-- คำเตือน: ลบข้อมูลทั้งหมดในตารางเหล่านี้ถาวร ใช้เฉพาะตอนตั้งค่าระบบครั้งแรก/ยังไม่มีข้อมูลจริง

DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS open_tabs CASCADE;
DROP TABLE IF EXISTS members CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS staff CASCADE;
DROP TABLE IF EXISTS order_sequences CASCADE;
DROP FUNCTION IF EXISTS next_order_id();
