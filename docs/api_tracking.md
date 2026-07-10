# Track Go API บนเครื่อง Server

คู่มือนี้ใช้ระบบ tracking ที่มีอยู่แล้วในโปรเจกต์:

- stdout request log จาก Fiber middleware สำหรับดู request สด
- usage log ลง PostgreSQL table `auth_logs.dmama_use` สำหรับ query ย้อนหลัง

## 1. ตั้งค่า Environment

ต้องตั้ง `GISDATA_URL` ให้ชี้ไป PostgreSQL 16 ที่ใช้เก็บ log:

```env
GISDATA_URL=postgres://dmama:password@10.250.230.81:5432/gis_data?sslmode=prefer
```

ถ้า `GISDATA_URL` ว่างหรือเชื่อมไม่ได้ server จะยังรัน endpoint หลักได้ แต่ usage logging จะถูกปิด และ log จะขึ้นข้อความประมาณ:

```text
WARN: GISDATA_URL not set; feature CRUD + usage logging disabled
```

## 2. รัน API บน Server

```powershell
go run .\cmd\server
```

เมื่อเริ่มสำเร็จจะเห็น:

```text
Server starting on port 5013
```

ทุก request จะมี stdout log รูปแบบ:

```text
2026-07-03 10:00:00 | 200 | 10ms | GET | /api/dma/stats
```

## 3. ขอบเขตที่ถูก Track

- ถูก track: request ใต้ `/api` ที่ผ่าน `X-API-Key` แล้ว
- ไม่ถูก track: `/api/health`
- ไม่ถูก track: request ที่ auth ไม่ผ่าน เช่นไม่มี `X-API-Key` ตอน `DMAMA_KEY` ถูกตั้งค่า

ระบบจะสร้าง schema/table ให้อัตโนมัติเมื่อ service start:

```sql
auth_logs.dmama_use
```

ข้อมูลที่เก็บหลัก ๆ คือ `api_key`, `method`, `path`, `status`, `size_bytes`, `size_value`, `size_unit`, `duration_ms`, `started_at`, `ended_at`, และ `request_id`.

## 4. Query สำหรับตรวจสอบ

ดู request ล่าสุด:

```sql
SELECT started_at, api_key, method, path, status, duration_ms, size_value, size_unit, request_id
FROM auth_logs.dmama_use
ORDER BY started_at DESC
LIMIT 100;
```

สรุปจำนวนเรียกตาม endpoint ใน 1 วันที่ผ่านมา:

```sql
SELECT path, method, status, count(*) AS calls, avg(duration_ms)::int AS avg_ms
FROM auth_logs.dmama_use
WHERE started_at >= now() - interval '1 day'
GROUP BY path, method, status
ORDER BY calls DESC;
```

ดูจำนวนตาม status code:

```sql
SELECT status, count(*) AS calls
FROM auth_logs.dmama_use
WHERE started_at >= now() - interval '1 day'
GROUP BY status
ORDER BY status;
```

ดู API key ที่ใช้งานเยอะสุด:

```sql
SELECT api_key, count(*) AS calls, sum(size_bytes) AS total_bytes, avg(duration_ms)::int AS avg_ms
FROM auth_logs.dmama_use
WHERE started_at >= now() - interval '1 day'
GROUP BY api_key
ORDER BY calls DESC;
```

## 5. วิธี Test หลัง Deploy

1. เรียก health check:

```powershell
Invoke-RestMethod http://localhost:5013/api/health
```

จากนั้น query `auth_logs.dmama_use` ต้องไม่พบ record ใหม่จาก `/api/health`.

2. เรียก endpoint ที่ต้องใช้ API key:

```powershell
Invoke-RestMethod -Headers @{"X-API-Key"="YOUR_KEY"} http://localhost:5013/api/dma/stats
```

จากนั้น query request ล่าสุด ต้องพบ record ของ endpoint ที่เรียก.

3. ทดสอบกรณี logging ปิด:

ตั้ง `GISDATA_URL` ผิดหรือเว้นว่าง แล้ว restart service. API ควรยังตอบ endpoint หลักได้ และ stdout log ต้องแจ้งว่า usage logging disabled.