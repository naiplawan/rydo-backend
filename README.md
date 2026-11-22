# RYDO Backend

โฟลเดอร์เก็บเอกสารผลตรวจ แผน และโค้ดเริ่มต้นของ RYDO Backend
ตอนนี้มี health endpoints และ pilot API 18 operations ที่ตรวจด้วย HTTP/ฐานข้อมูลทดสอบแล้ว ยังไม่ตรวจ LINE จริงหรือเก็บข้อมูลจริง

- [ผลตรวจ local pilot และวิธีเชื่อม frontend](verification/local-pilot/REPORT.md)
- [ตั้งค่า Railway และ GitHub Actions](DEPLOYMENT.md)
- [Task.md](Task.md): รายการงานพร้อม dependency เกณฑ์รับงาน และลิงก์ย้อนถึง SRS/PRD
- [SRS ฉบับร่าง](SRS.md): รหัสข้อกำหนด Backend ที่รวบรวมจาก PRD และสเปกเดิม พร้อมลิงก์กลับสู่งาน
- [ผลตรวจ Backend](BACKEND_REVIEW.md): สถานะที่ตรวจพบ ช่องว่าง และผลตรวจสูตร
- [แผนพัฒนาและเกณฑ์ตรวจรับ](IMPLEMENTATION_PLAN.md): แยกต้นแบบทดลอง 7 วันออกจาก MVP หลังตรวจร่วมกับ Tandem พร้อมลำดับงานและวิธีตรวจ

สแต็กตามเอกสารโครงการ: Go + Gin, PostgreSQL + sqlc, Redis + Asynq
เอกสารในโฟลเดอร์นี้เป็นผลตรวจและข้อเสนอ ไม่เปลี่ยนสัญญา API หรือขอบเขต MVP เดิม

## เอกสารต้นทาง

- [PRD](../rydo-doc/01_PRD.md)
- [สถาปัตยกรรม](../rydo-doc/02_SYSTEM_ARCHITECTURE.md)
- [ฐานข้อมูล](../rydo-doc/03_DATABASE_SCHEMA.md)
- [สเปก API](../rydo-doc/04_API_SPECIFICATION.md)
- [Backlog](../rydo-doc/05_IMPLEMENTATION_BACKLOG.md)
- [เกณฑ์ทดลองและตัดสินใจพัฒนา](../rydo-doc/06_PRODUCT_DECISIONS_AND_RISKS.md)
- [การตั้งค่า LINE](../rydo-doc/07_LINE_SETUP.md)
- [งานวิจัยวิธีคำนวณ](../rydo-doc/research/rider-profit-per-hour.md)

อ้างอิงเอกสารต้นทางโดยตรงเพื่อไม่สร้างสำเนาสเปกที่อาจแก้ไขไม่ตรงกัน

## รัน Backend ในเครื่อง

ใช้ Go 1.25 ขึ้นไป, PostgreSQL ที่มี `gen_random_uuid()` ในตัว และ golang-migrate ตั้ง `DATABASE_URL` ให้ชี้ไปยังฐานข้อมูลสำหรับพัฒนา

ก่อนรัน ต้องตั้ง `WEB_APP_URL`, `LINE_LOGIN_CHANNEL_ID`, `JWT_SIGNING_KEY` และ `PREVIEW_SIGNING_KEY` ตาม `.env.example`
keys ต้องเป็นคนละค่าสุ่มอย่างน้อย 32 ไบต์ ไม่มี default หรือ auth bypass; ไม่ใส่ค่าจริงลง source หรือ log
API ตรวจ LINE ID token ผ่าน endpoint จริงแบบ fixed URL ส่วน test double มีเฉพาะใน `_test.go`

```sh
migrate -path db/migrations -database "$DATABASE_URL" up
go run -buildvcs=false ./cmd/rydo-api
```

API ใช้ `PORT` ถ้ามีค่า จากนั้นใช้ `APP_PORT` และ 8080 ตามลำดับ
`GET /health/live` ตรวจว่า HTTP server ทำงาน ส่วน `GET /health/ready` ตรวจว่าเชื่อม PostgreSQL ได้และ migration ถึงเวอร์ชันที่ API ต้องการโดยไม่มีสถานะ dirty
หากแยก role ของ API ออกจาก role ที่รัน migration ให้ role ของ API อ่านตาราง `schema_migrations` ได้

เปิด Swagger UI ที่ [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
หรืออ่าน OpenAPI JSON ที่ [http://localhost:8080/swagger/openapi.json](http://localhost:8080/swagger/openapi.json)
เอกสารนี้แสดงสอง health endpoints และ 18 pilot operations ที่ผ่าน HTTP/DB contract checks; สัญญา proposed เดิมยังแยกอยู่ใน `contracts/t01/`

```sh
go vet ./...
go test ./...
go build -buildvcs=false ./...
```

รัน `sh scripts/test-foundation.sh` ด้วยผู้ใช้ทั่วไปที่มีเครื่องมือ PostgreSQL server และ client, `curl` และ `migrate` เพื่อตรวจ readiness และ migration บนฐานข้อมูลชั่วคราวในเครื่อง


## ตรวจ local pilot ซ้ำ

`go test ./...` เริ่ม PostgreSQL cluster ส่วนตัวใน temporary directory (Unix socket, ไม่มี TCP)
และรัน migrations เอง ไม่อ่าน `DATABASE_URL` มาใช้เป็นฐานข้อมูลทดสอบ ต้องมี `pg_config`, `initdb`, `pg_ctl`
และรันด้วยผู้ใช้ทั่วไป จากนั้นปิด cluster และลบเฉพาะข้อมูลชั่วคราวที่ test สร้าง

```sh
go vet ./...
RYDO_CAPTURE_HTTP="$PWD/verification/local-pilot/http-capture.json" go test -count=1 ./...
go test -race -count=1 ./...
go build -buildvcs=false ./...
uv run --with openapi-spec-validator==0.9.0 --with jsonschema==4.26.0 python scripts/validate-pilot-http.py
```

HTTP capture เป็นข้อมูลสมมติและลบ token ออกจากหลักฐานแล้ว คำสั่งนี้ไม่ทดสอบบัญชี LINE จริง
การทดสอบใน LINE/browser จริงยังต้องมี LIFF ID, HTTPS และ same-origin proxy ที่ส่ง Origin/Set-Cookie ถูกต้อง
ห้ามลด Secure ของ cookie หรือเพิ่ม fake-login route เพื่อให้ผ่านการเชื่อมจริง
วันที่และ vehicle ของรายการที่บันทึกแล้วยัง immutable; DELETE ไม่ใช่ workflow แก้วันที่
มี workflow CI/CD สำหรับ GitHub Actions แล้ว ส่วน sqlc และ T09/T10 review/LINE readiness ยังต้องให้เจ้าของงานปิดตามแผนก่อนข้อมูลจริง
