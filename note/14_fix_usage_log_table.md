# Fix Usage Log Table: auth_logs.dmama_use

## ปัญหา

ตอนเรียก API แล้ว query usage log พบ error:

```text
ERROR: relation "auth_logs.dmama_use" does not exist
```

สาเหตุที่เป็นไปได้สูงคือ database ที่ `GISDATA_URL` ชี้ไปยังไม่ได้สร้าง table
`auth_logs.dmama_use` หรือ service ใช้ role ที่ไม่มีสิทธิ์สร้าง schema/table ตอน start.

## วิธีแก้บน Server

รัน migration นี้บน PostgreSQL database เดียวกับ `GISDATA_URL` ไม่ใช่ `DATABASE_URL`:

```sql
\i migrations/0002_auth_logs_dmama_use.sql
```

ไฟล์ migration จะสร้าง:

- schema `auth_logs`
- table `auth_logs.dmama_use`
- indexes `dmama_use_api_key_idx`, `dmama_use_started_at_idx`
- grant ให้ role `dmama` สำหรับ insert/select usage log

## ตรวจสอบหลังรัน Migration

ตรวจว่า table มีอยู่:

```sql
SELECT to_regclass('auth_logs.dmama_use');
```

ควรได้ผลลัพธ์:

```text
auth_logs.dmama_use
```

จากนั้น restart service แล้วเรียก endpoint ที่ต้องใช้ API key เช่น:

```powershell
Invoke-RestMethod -Headers @{"X-API-Key"="YOUR_KEY"} http://localhost:5013/api/dma/stats
```

ตรวจ log ล่าสุด:

```sql
SELECT started_at, api_key, method, path, status, duration_ms, size_value, size_unit, request_id
FROM auth_logs.dmama_use
ORDER BY started_at DESC
LIMIT 5;
```

ต้องเห็น record ใหม่จาก endpoint ที่เรียก.

## หมายเหตุสำหรับ App

โค้ดใน `UsageRecorder.EnsureSchema` จะตรวจ table ตอน start ก่อน ถ้า table พร้อมแล้วจะไม่ยิง DDL.
ถ้า table ไม่มีและ app role มีสิทธิ์ DDL จะพยายามสร้างให้แบบ fallback.
ถ้าสร้างไม่ได้ error จะบอกให้รัน `migrations/0002_auth_logs_dmama_use.sql` บน `GISDATA_URL`.
