// ============================================================
// api.js — ตัวแทน google.script.run เดิม แต่ยิงไปหา Go backend จริง
// ทุกฟังก์ชันตั้งชื่อเหมือนฟังก์ชันเดิมทุกตัว และคืนค่าผลลัพธ์รูปร่างเดียวกับของเดิม
// เพื่อให้โค้ดส่วนที่เหลือของ index.html แก้น้อยที่สุด (แค่เปลี่ยนวิธีเรียก)
// ============================================================

const API_BASE_URL = window.API_BASE_URL;
const TOKEN_KEY = "pixelpos_token";
const STAFF_KEY = "pixelpos_staff";

function getToken() { return localStorage.getItem(TOKEN_KEY); }
function setToken(t) { localStorage.setItem(TOKEN_KEY, t); }
function clearToken() { localStorage.removeItem(TOKEN_KEY); }

function getStaff() {
  try { return JSON.parse(localStorage.getItem(STAFF_KEY)); } catch (e) { return null; }
}
function setStaff(staff) { localStorage.setItem(STAFF_KEY, JSON.stringify(staff)); }
function clearStaff() { localStorage.removeItem(STAFF_KEY); }

// เรียกจากทุกหน้าที่ต้อง login ก่อน (pos.html, products.html, reports.html)
// ถ้ายังไม่ login จะเด้งกลับ login.html ให้อัตโนมัติ คืนค่า staff object ถ้า login อยู่แล้ว
function requireAuth() {
  const token = getToken();
  const staff = getStaff();
  if (!token || !staff) {
    location.href = "login.html";
    return null;
  }
  const el = document.getElementById("staff-name");
  if (el) el.innerText = "พนักงาน: " + staff.name;
  return staff;
}

function logout() {
  clearToken();
  clearStaff();
  location.href = "login.html";
}

// apiCall ไม่ throw/reject เวลาเจอ error ธุรกิจ (เช่น validation ไม่ผ่าน) — จะ resolve เป็น
// { success:false, message: "..." } เหมือนพฤติกรรมเดิมของ google.script.run เสมอ
// จะ reject จริงๆ เฉพาะตอน token หมดอายุ (401) ซึ่งจะ auto กลับไปหน้า login ให้เลย
async function apiCall(method, path, body, needsAuth = true) {
  const headers = { "Content-Type": "application/json" };
  if (needsAuth) headers["Authorization"] = "Bearer " + getToken();

  let res;
  try {
    res = await fetch(API_BASE_URL + path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch (e) {
    return { success: false, message: "เชื่อมต่อเซิร์ฟเวอร์ไม่ได้ กรุณาเช็คอินเทอร์เน็ต" };
  }

  if (res.status === 401 && needsAuth) {
    clearToken();
    Swal.fire('เซสชันหมดอายุ', 'กรุณาเข้าสู่ระบบใหม่อีกครั้ง', 'warning').then(() => {
      location.reload();
    });
    return new Promise(() => {}); // ค้างไว้เฉยๆ เพราะกำลังจะ reload หน้าอยู่แล้ว
  }

  try {
    return await res.json();
  } catch (e) {
    return { success: false, message: "เซิร์ฟเวอร์ตอบกลับผิดรูปแบบ" };
  }
}

// --- helper วันที่ (เวลาไทย) ---
function todayThaiISO() {
  const thai = new Date(Date.now() + 7 * 60 * 60 * 1000);
  return thai.toISOString().slice(0, 10); // YYYY-MM-DD
}
function isoToThaiSlash(iso) {
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
}

const api = {
  // ---------- auth ----------
  checkLogin: (empId) =>
    apiCall("POST", "/api/auth/login", { emp_id: empId }, false).then((r) => {
      if (r.success) {
        setToken(r.token);
        setStaff({ name: r.name, empId: r.emp_id, role: r.role });
      }
      return { success: r.success, message: r.message, name: r.name, empId: r.emp_id, role: r.role };
    }),

  // ---------- products ----------
  getProducts: () =>
    apiCall("GET", "/api/products").then((r) =>
      r.success
        ? r.data.map((p) => ({
            id: p.id,
            barcode: p.barcode || "",
            name: p.name,
            category: p.category || "",
            price: p.price,
            image: p.image_url || "",
            status: p.status,
            stock: p.stock === null || p.stock === undefined ? "" : p.stock,
          }))
        : []
    ),

  addNewProduct: (barcode, name, category, price, image) =>
    apiCall("POST", "/api/products", {
      barcode: barcode || "",
      name,
      category,
      price: parseFloat(price) || 0,
      image_url: image || "",
    }).then((r) => ({ success: r.success, message: r.message, id: r.id })),

  // อัปโหลดไฟล์รูปจริง (ฟังก์ชันใหม่ที่ระบบเดิมไม่มี — เดิมกรอกแค่ URL รูป)
  // ใช้ FormData แทน JSON เพราะเป็นการส่งไฟล์ ห้ามตั้ง Content-Type เองต้องให้ browser ตั้ง boundary ให้อัตโนมัติ
  uploadProductImage: (id, file) => {
    const formData = new FormData();
    formData.append("image", file);
    return fetch(`${API_BASE_URL}/api/products/${id}/image`, {
      method: "POST",
      headers: { Authorization: "Bearer " + getToken() },
      body: formData,
    })
      .then((res) => res.json())
      .catch(() => ({ success: false, message: "อัปโหลดรูปไม่สำเร็จ กรุณาลองใหม่" }));
  },

  // ใช้ id แทนชื่อเดิม (oldName) — ของเดิม: updateProductDetails(oldName, newData)
  updateProductDetails: (id, newData) =>
    apiCall("PUT", `/api/products/${id}`, {
      barcode: newData.barcode || "",
      name: newData.name,
      category: newData.category,
      price: parseFloat(newData.price) || 0,
      image_url: newData.image || "",
    }),

  // ใช้ id แทนชื่อเดิม — ของเดิม: toggleProductStatus(name, currentStatus)
  toggleProductStatus: (id) => apiCall("PATCH", `/api/products/${id}/status`, {}),

  // ใช้ id แทนชื่อเดิม — ของเดิม: updateProductStock(name, newStock)
  updateProductStock: (id, newStock) =>
    apiCall("PATCH", `/api/products/${id}/stock`, {
      stock: newStock === "" || newStock === null || newStock === undefined ? null : parseInt(newStock, 10),
    }),

  // ใช้ id แทนชื่อเดิม — ของเดิม: deleteProduct(name)
  deleteProduct: (id) => apiCall("DELETE", `/api/products/${id}`),

  // ---------- transactions ----------
  saveTransaction: (cartStr, total, cash, transfer, empInfo, remark, cashReceived, changeAmount, memberPhone) =>
    apiCall("POST", "/api/transactions", {
      cart: JSON.parse(cartStr),
      total,
      cash,
      transfer,
      remark,
      cash_received: cashReceived,
      change_amount: changeAmount,
      member_phone: memberPhone || "",
    }).then((r) => ({ success: r.success, message: r.message, orderId: r.order_id })),

  getTransactionHistory: (targetDate) =>
    apiCall("GET", "/api/transactions" + (targetDate ? `?date=${targetDate}` : "")).then((r) => {
      if (!r.success) return r;
      return {
        success: true,
        data: r.data.map((t) => ({
          orderId: t.order_id,
          time: t.time,
          total: t.total,
          status: t.status,
          empInfo: t.emp_info,
          cartStr: JSON.stringify(t.cart),
        })),
      };
    }),

  // ของเดิม: cancelBill(orderId, empId) — empId ไม่ต้องส่งแล้วเพราะ backend รู้จากตัว JWT เอง
  cancelBill: (orderId) => apiCall("POST", `/api/transactions/${orderId}/cancel`, {}),

  getDailySummary: (targetDate) => {
    const rawDate = targetDate || todayThaiISO();
    const dateStr = isoToThaiSlash(rawDate);
    return apiCall("GET", `/api/reports/daily?date=${rawDate}`).then((r) => {
      if (!r.success) return r;
      const itemData = [["สินค้า", "จำนวน", "ยอดรวม"]].concat(
        (r.data.items || []).map((it) => [it.name, it.qty, it.total])
      );
      return {
        success: true,
        rawDate,
        dateStr,
        data: { bills: r.data.bills, total: r.data.total, cash: r.data.cash, transfer: r.data.transfer, itemData },
      };
    });
  },

  // ---------- open tabs (พักบิล) ----------
  saveOpenTab: (tabName, cartStr, remark) =>
    apiCall("POST", "/api/open-tabs", { tab_name: tabName, cart: JSON.parse(cartStr), remark: remark || "" }),

  getOpenTabs: () =>
    apiCall("GET", "/api/open-tabs").then((r) =>
      r.success
        ? r.data.map((t) => ({
            row: t.tab_id,
            tabName: t.tab_name,
            cartData: JSON.stringify(t.cart),
            timestamp: t.created_at,
          }))
        : []
    ),

  deleteOpenTab: (tabId) => apiCall("DELETE", `/api/open-tabs/${tabId}`),

  // ---------- members ----------
  getMemberByPhone: (phone) =>
    apiCall("GET", `/api/members/${phone}`).then((r) =>
      r.success
        ? { success: true, phone: r.phone, name: r.name, points: r.points, grade: r.grade, lastClaimMonth: r.last_claim_month }
        : r
    ),

  registerNewMember: (phone, name) => apiCall("POST", "/api/members", { phone, name }),

  redeemPoints: (phone, pointsDeduct, rewardName) =>
    apiCall("POST", `/api/members/${phone}/redeem`, { points: pointsDeduct, reward_name: rewardName }).then((r) => ({
      success: r.success,
      message: r.message,
      newPoints: r.new_points,
    })),

  claimMonthlyFreebie: (phone, month) =>
    apiCall("POST", `/api/members/${phone}/claim-freebie`, { month }),
};
