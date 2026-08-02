package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// Client คุยกับ Supabase Storage REST API โดยตรง (ไม่ต้องพึ่ง SDK)
// ใช้ service_role key เท่านั้น เพราะต้องมีสิทธิ์เขียนไฟล์เข้า bucket
// ห้ามเอา service_role key ไปฝังใน frontend เด็ดขาด ต้องผ่าน backend เท่านั้น
type Client struct {
	BaseURL    string // เช่น https://xxxxx.supabase.co
	ServiceKey string
	Bucket     string
	httpClient *http.Client
}

func NewClient(baseURL, serviceKey, bucket string) *Client {
	return &Client{BaseURL: baseURL, ServiceKey: serviceKey, Bucket: bucket, httpClient: &http.Client{}}
}

// UploadImage อัปโหลดไฟล์เข้า bucket แล้วคืน public URL กลับมา
// ใช้ "upsert": true เพื่อให้อัปโหลดทับชื่อไฟล์เดิมได้เลยถ้าซ้ำ (กันไฟล์ขยะสะสมตอนแก้รูปซ้ำๆ)
func (c *Client) UploadImage(objectPath string, data []byte, contentType string) (string, error) {
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.BaseURL, c.Bucket, objectPath)

	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.ServiceKey)
	req.Header.Set("apikey", c.ServiceKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase storage upload ล้มเหลว (status %d): %s", resp.StatusCode, string(body))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.BaseURL, c.Bucket, objectPath)
	return publicURL, nil
}
