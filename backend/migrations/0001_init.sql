-- ============ staff ============
CREATE TABLE staff (
  emp_id      TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  role        TEXT NOT NULL DEFAULT 'staff',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ products ============
CREATE TABLE products (
  id          BIGSERIAL PRIMARY KEY,
  barcode     TEXT UNIQUE,
  name        TEXT NOT NULL,
  category    TEXT,
  price       NUMERIC(10,2) NOT NULL DEFAULT 0,
  image_url   TEXT,
  status      TEXT NOT NULL DEFAULT 'เปิด',   -- 'เปิด' | 'ปิด'
  stock       INTEGER,                        -- NULL = ขายไม่จำกัด
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_barcode ON products(barcode);
CREATE INDEX idx_products_name    ON products(name);

-- ============ members ============
CREATE TABLE members (
  phone             TEXT PRIMARY KEY,
  name              TEXT NOT NULL,
  points            INTEGER NOT NULL DEFAULT 0,
  lifetime_total    NUMERIC(12,2) NOT NULL DEFAULT 0,
  grade             SMALLINT NOT NULL DEFAULT 0,   -- 0-4
  last6mo_total     NUMERIC(12,2) NOT NULL DEFAULT 0,
  last_active_date  DATE,
  last_claim_month  TEXT,                          -- 'YYYY-MM'
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ transactions ============
CREATE TABLE transactions (
  order_id        TEXT PRIMARY KEY,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  cart            JSONB NOT NULL,
  total           NUMERIC(12,2) NOT NULL,
  cash            NUMERIC(12,2) NOT NULL DEFAULT 0,
  cash_received   NUMERIC(12,2) NOT NULL DEFAULT 0,
  change_amount   NUMERIC(12,2) NOT NULL DEFAULT 0,
  transfer        NUMERIC(12,2) NOT NULL DEFAULT 0,
  emp_info        TEXT,
  remark          TEXT,
  member_phone    TEXT REFERENCES members(phone),
  status          TEXT NOT NULL DEFAULT 'ปกติ',   -- 'ปกติ' | 'ยกเลิก'
  cancelled_by    TEXT,
  cancelled_at    TIMESTAMPTZ
);
CREATE INDEX idx_tx_created_at ON transactions(created_at);
CREATE INDEX idx_tx_status     ON transactions(status);

-- ============ open_tabs (พักบิล) ============
CREATE TABLE open_tabs (
  tab_id      TEXT PRIMARY KEY,
  tab_name    TEXT NOT NULL,
  cart        JSONB NOT NULL,
  remark      TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ ตัวช่วยออกเลขบิลแบบ atomic ============
CREATE TABLE order_sequences (
  day_key   TEXT PRIMARY KEY,  -- 'yyyyMMdd'
  last_seq  INTEGER NOT NULL DEFAULT 0
);

CREATE OR REPLACE FUNCTION next_order_id() RETURNS TEXT AS $$
DECLARE
  d TEXT := to_char(now() AT TIME ZONE 'Asia/Bangkok', 'YYYYMMDD');
  s INTEGER;
BEGIN
  INSERT INTO order_sequences(day_key, last_seq) VALUES (d, 1)
    ON CONFLICT (day_key) DO UPDATE SET last_seq = order_sequences.last_seq + 1
    RETURNING last_seq INTO s;
  RETURN 'INV' || d || '-' || LPAD(s::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;
