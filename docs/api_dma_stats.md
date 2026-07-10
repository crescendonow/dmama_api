# API Document: DMA Stats

เอกสารนี้สรุปการเรียกใช้งาน API สำหรับดึงสถิติการใช้น้ำและจำนวนผู้ใช้น้ำใน DMA เดียว เพื่อส่งต่อให้ทีม developer

## Endpoint

```http
GET https://gisweb2.pwa.co.th/dmama_api/api/dma/stats
```

Production reverse proxy `/dmama_api/` forward ไปยัง Go service ที่ route ภายในเป็น:

```http
GET /api/dma/stats
```

## Authentication

ต้องส่ง API key ผ่าน header:

```http
X-API-Key: <YOUR_API_KEY>
```

ถ้าไม่ส่ง key หรือ key ไม่ถูกต้อง จะได้ `401`.

```json
{
  "success": false,
  "error": "invalid or missing API key"
}
```

หมายเหตุ: ใน dev mode ถ้า environment `DMAMA_KEY` ว่าง ระบบจะ skip authentication แต่ production endpoint ต้องใช้ key

## Method และ Payload

API นี้ใช้ `GET` และไม่มี request body payload

ส่ง input ผ่าน query string เท่านั้น:

```http
GET /api/dma/stats?pwa_code=5531011&dma_id=1&column=prswtusg
```

## Query Parameters

| Parameter | Required | Type | Example | รายละเอียด |
|---|---:|---|---|---|
| `pwa_code` | Yes | string | `5531011` | รหัส กปภ.สาขา/พื้นที่ ใช้ค้นหา DMA และใช้ 4 หลักแรกเพื่อหา region ถ้าไม่ส่ง `region` |
| `dma_id` | Yes | string | `1` | รหัส DMA ใน `pwa_dma.dma_boundary` |
| `region` | No | integer | `1` | เขตข้อมูล 1-10 ถ้าส่งมา ระบบจะใช้ค่านี้แทนการ infer จาก `pwa_code` |
| `column` | No | string | `prswtusg` | column ปริมาณน้ำที่ต้องการ sum ค่า default คือ `prswtusg` |
| `year` | No | integer | `2026` | ใช้คู่กับ `month` เพื่อให้ระบบแปลงเป็น `prswtusg` หรือ `lstwtusgN` อัตโนมัติ |
| `month` | No | integer 1-12 | `6` | ต้องส่งคู่กับ `year` เท่านั้น |

### Region Mapping จาก pwa_code

ถ้าไม่ส่ง `region` ระบบจะอ่าน 4 หลักแรกของ `pwa_code` เพื่อเลือก region/table:

| Region | pwa_code prefix |
|---:|---|
| 1 | `5531` |
| 2 | `5541` |
| 3 | `5542` |
| 4 | `5551` |
| 5 | `5552` |
| 6 | `5521` |
| 7 | `5522` |
| 8 | `5532` |
| 9 | `5511` |
| 10 | `5512` |

ถ้า prefix ไม่อยู่ใน mapping จะได้ `400`.

### Allowed `column`

API validate column ด้วย allowlist เพื่อป้องกัน SQL injection:

```text
prswtusg
use_water
use_jan, use_feb, use_mar, use_apr, use_may, use_jun,
use_jul, use_aug, use_sep, use_oct, use_nov, use_dec
lstwtusg1, lstwtusg2, lstwtusg3, lstwtusg4, lstwtusg5, lstwtusg6,
lstwtusg7, lstwtusg8, lstwtusg9, lstwtusg10, lstwtusg11, lstwtusg12
```

ถ้าไม่ส่ง `column` จะใช้ `prswtusg`.

### การใช้ `year` และ `month`

ถ้าส่ง `year` หรือ `month` ต้องส่งครบทั้งคู่ และระบบจะ ignore `column` แล้วแปลงเดือนที่ระบุเป็น column ให้อัตโนมัติ:

| เงื่อนไข | Column ที่ใช้ |
|---|---|
| เดือนที่ขอเป็นรอบบิลล่าสุด | `prswtusg` |
| ย้อนหลัง 1 เดือน | `lstwtusg1` |
| ย้อนหลัง 2 เดือน | `lstwtusg2` |
| ย้อนหลังมากกว่า 12 เดือน | clamp เป็น `lstwtusg12` |

กฎรอบบิลในโค้ด:

- ใช้วันที่ปัจจุบันของ server เป็นตัวคำนวณ
- ถ้าวันที่ปัจจุบันน้อยกว่า 20 จะถือว่ารอบบิลล่าสุดยังไม่ปิด และลด diff month ลง 1
- ถ้า request เดือนที่ใหม่กว่ารอบบิลล่าสุด จะได้ `400`

## Example Requests

### 1. เรียกด้วย column ปัจจุบัน

```bash
curl -H "X-API-Key: YOUR_KEY" \
  "https://gisweb2.pwa.co.th/dmama_api/api/dma/stats?pwa_code=5531011&dma_id=1&column=prswtusg"
```

### 2. ให้ระบบหา region จาก pwa_code

```bash
curl -H "X-API-Key: YOUR_KEY" \
  "https://gisweb2.pwa.co.th/dmama_api/api/dma/stats?pwa_code=5531011&dma_id=1"
```

### 3. ระบุ region เอง

```bash
curl -H "X-API-Key: YOUR_KEY" \
  "https://gisweb2.pwa.co.th/dmama_api/api/dma/stats?pwa_code=5531011&dma_id=1&region=1&column=prswtusg"
```

### 4. เรียกด้วย year/month

```bash
curl -H "X-API-Key: YOUR_KEY" \
  "https://gisweb2.pwa.co.th/dmama_api/api/dma/stats?pwa_code=5531011&dma_id=1&year=2026&month=6"
```

## Success Response

HTTP status: `200 OK`

```json
{
  "success": true,
  "data": {
    "pwa_code": "5531011",
    "dma_id": "1",
    "column": "prswtusg",
    "usage": {
      "total": 164120,
      "house": 106181,
      "government": 575,
      "business_small": 34630,
      "business_large": 15332
    },
    "population": {
      "total": 7816,
      "house": 6714,
      "government": 5,
      "business_small": 870,
      "business_large": 227
    }
  }
}
```

ตัวอย่างนี้ยืนยันจาก production endpoint วันที่ 2026-07-03 โดยใช้ request:

```http
GET /dmama_api/api/dma/stats?pwa_code=5531011&dma_id=1&column=prswtusg
```

latency ที่วัดจากเครื่องทดสอบครั้งเดียวประมาณ `245 ms` จึงควรมองเป็นตัวอย่าง ไม่ใช่ SLA

## Response Fields

| Field | Type | รายละเอียด |
|---|---|---|
| `success` | boolean | `true` เมื่อสำเร็จ |
| `data.pwa_code` | string | pwa_code ที่ request |
| `data.dma_id` | string | dma_id ที่ request |
| `data.column` | string | column ที่ถูกใช้จริง หลัง resolve จาก `column` หรือ `year/month` |
| `data.usage.total` | number | ผลรวมปริมาณน้ำของลูกค้าทุกประเภทใน DMA |
| `data.usage.house` | number | ผลรวม usage ของ `usetype` กลุ่มบ้าน: `11,12,13,14,15` |
| `data.usage.government` | number | ผลรวม usage ของ `usetype` กลุ่มราชการ: `21,22,24,25,27` |
| `data.usage.business_small` | number | ผลรวม usage ของ `usetype` กลุ่มธุรกิจขนาดเล็ก: `23,26,28,29` |
| `data.usage.business_large` | number | ผลรวม usage ของ `usetype` ที่ขึ้นต้นด้วย `3` |
| `data.population.total` | integer | จำนวน customer ที่ column มีค่า `> -1` |
| `data.population.house` | integer | จำนวน customer บ้านที่ column มีค่า `> -1` |
| `data.population.government` | integer | จำนวน customer ราชการที่ column มีค่า `> -1` |
| `data.population.business_small` | integer | จำนวน customer ธุรกิจขนาดเล็กที่ column มีค่า `> -1` |
| `data.population.business_large` | integer | จำนวน customer ธุรกิจขนาดใหญ่ที่ column มีค่า `> -1` |

## Error Responses

### 400 Bad Request

เกิดจาก parameter ไม่ครบหรือไม่ถูกต้อง

```json
{
  "success": false,
  "error": "pwa_code and dma_id are required"
}
```

ข้อความ error ที่พบได้:

| Case | Error |
|---|---|
| ไม่ส่ง `pwa_code` และไม่มี `region` | `region or pwa_code is required` |
| ส่ง `pwa_code` แต่ไม่ส่ง `dma_id` | `pwa_code and dma_id are required` |
| `region` ไม่ใช่ 1-10 | `region must be 1-10` |
| `pwa_code` สั้นกว่า 4 หลัก | `invalid pwa_code: too short` |
| prefix ของ `pwa_code` ไม่อยู่ใน mapping | `unknown pwa_code prefix: <prefix>` |
| `column` ไม่อยู่ใน allowlist | `invalid column: <column>` |
| ส่ง `year` หรือ `month` มาแค่ตัวเดียว | `year and month must be provided together` |
| `year` ไม่ใช่ตัวเลข | `year must be a number` |
| `month` ไม่ใช่ 1-12 | `month must be 1-12` |
| เดือนที่ request ใหม่กว่ารอบบิลล่าสุด | `requested month is newer than the latest completed billing cycle` |

### 401 Unauthorized

```json
{
  "success": false,
  "error": "invalid or missing API key"
}
```

### 404 Not Found

ถ้าไม่พบ DMA ใน `pwa_dma.dma_boundary`:

```json
{
  "success": false,
  "error": "DMA not found for pwa_code=5531011, dma_id=999"
}
```

### 500 Internal Server Error

เกิดจาก database/query/runtime error:

```json
{
  "success": false,
  "error": "<database or server error message>"
}
```

## Data Source และ Logic Summary

API นี้ query จาก:

- DMA boundary: `pwa_dma.dma_boundary`
- Customer table ตาม region: `giswebm_stamp.r{region}_bl_customer`

Join logic:

- `LEFT JOIN` customer กับ DMA boundary ด้วย `pwa_code`
- ใช้ spatial condition `ST_Intersects(dma.wkb_geometry, bl.wkb_geometry)`
- ถ้า SRID ของ customer geometry ไม่ตรงกับ DMA geometry จะ set/transform SRID ก่อน intersect
- filter DMA ด้วย `dma.pwa_code = $1` และ `dma.dma_id = $2`

ผลลัพธ์เป็น aggregate ของ DMA เดียว ไม่มี geometry และไม่มีรายการ customer รายตัว

## Timeout

จากโค้ด Go service ปัจจุบัน:

- ยังไม่ได้ตั้งค่า Fiber `ReadTimeout`, `WriteTimeout`, หรือ `IdleTimeout` เฉพาะใน application
- ยังไม่มี per-request context timeout ใน handler/service/repository ของ `/api/dma/stats`
- database pool ตั้งค่า `MaxConns = 20`, `MinConns = 5`

จาก `nginx.conf` ที่พบใน repo:

- global `client_header_timeout 15s`
- global `client_body_timeout 15s`
- global `send_timeout 10s`
- global `keepalive_timeout 30s`
- location `/dmama_api/` proxy ไป `http://localhost:5013/`
- location `/dmama_api/` ไม่ได้ตั้ง `proxy_read_timeout`, `proxy_connect_timeout`, `proxy_send_timeout` เฉพาะใน block นี้ ดังนั้น production จริงจะขึ้นกับค่า default ของ nginx หรือ config อื่นที่ include เพิ่ม

คำแนะนำฝั่ง client:

- ตั้ง client timeout อย่างน้อย `30s`
- ถ้าต้องเรียกหลาย DMA ต่อเนื่อง ให้ retry เฉพาะกรณี network/timeout/5xx และใช้ backoff
- ไม่ควร retry ทันทีแบบถี่ ๆ เพราะ query เป็น spatial aggregate ที่อาจใช้ database resource สูง

## Data Limit

API นี้ไม่มี parameter `limit`, `offset`, หรือ pagination เพราะ response เป็น aggregate ของ DMA เดียว

ข้อจำกัดที่สำคัญ:

- 1 request = 1 `pwa_code` + 1 `dma_id`
- response เป็น object เดียว ไม่ใช่ list
- ไม่มี hard limit จำนวน customer ที่ scan ใน request จาก application code
- ปริมาณข้อมูลที่ query ขึ้นกับจำนวน customer point ที่ spatially intersect กับ DMA boundary นั้น
- request body ไม่มี จึงไม่ควรส่ง body
- reverse proxy มี global `client_max_body_size 16M` แต่ endpoint นี้เป็น `GET` และไม่ใช้ body

ถ้าต้องการดึงหลาย DMA ใน region เดียว ควรพิจารณาใช้ endpoint อื่นที่ออกแบบสำหรับ region aggregate เช่น `/api/dma/stats-region` แทนการยิง `/api/dma/stats` ซ้ำจำนวนมาก

## Cache Behavior

ฝั่ง service มี in-memory cache สำหรับ `/api/dma/stats`:

- TTL: `24 hours`
- cache key: `region:pwa_code:dma_id:column`
- request พร้อมกันที่ key เดียวกันจะถูก deduplicate ด้วย singleflight
- cache อยู่ใน memory ของ process เดียว ถ้า restart service หรือมีหลาย instance cache จะไม่ shared กัน

ผลกระทบ:

- การเรียกซ้ำด้วย parameter เดิมจะเร็วขึ้นและได้ข้อมูล cache ภายใน 24 ชั่วโมง
- ถ้าข้อมูล customer ใน database update แล้ว อาจยังเห็นค่าเก่าจนกว่า cache หมดอายุหรือ service restart

## Developer Notes

- ใช้ `column` จาก response เป็น source of truth ว่าระบบใช้เดือน/column ใดจริง
- ห้าม concatenate column เองนอก allowlist
- สำหรับ frontend ควรแสดง error จาก field `error` เมื่อ `success=false`
- สำหรับ backend integration ควร log `pwa_code`, `dma_id`, `region`, `column/year/month`, HTTP status, latency แต่ไม่ควร log `X-API-Key`
