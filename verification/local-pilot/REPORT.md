# Backend local-pilot implementation — T03–T08

วันที่: 2026-11-21T23:41:55+07:00
สถานะ: **Implemented and locally verified; awaiting Rika review / real LINE integration**
Source manifest SHA-256: `f07758b928e1a7e1b17ea3d37f858a6a6f2e9416561c35a5f2ca54eea45147d7`
โฟลเดอร์ยังไม่มี Git repository จึงไม่มี branch/commit ไม่ได้ commit/push/deploy

## ผลลัพธ์

พัฒนา API ภายใต้ `internal/pilot/` และประกอบเข้ากับ executable `cmd/rydo-api`
คง Go/Gin, pgx/PostgreSQL, migration version3 และสูตรใน `internal/profit/` เดิม
ไม่แก้ frontend, สเปก API/PRD ต้นทาง หรือ migrations
ยืนยัน assignment ที่ให้ Ferris เป็นเจ้าของ Backend; ตรวจ process/ไฟล์ก่อนเริ่ม ไม่พบผู้เขียน Backend อื่นทำงานอยู่ในรอบนี้
T02 foundation ที่มีอยู่ถูกใช้ต่อ ไม่สร้างฐานระบบหรือ migration ซ้ำ

## Endpoints ที่ implement และผ่าน HTTP/DB จริง

ทั้งหมดขึ้นต้น `/api/v1` ยกเว้น health/Swagger:

| Method | Path | ผลที่ตรวจ |
|---|---|---|
| POST | /auth/line | verified subject→user/session; reject invalid identity/origin; secure refresh cookie |
| POST | /auth/refresh | original30-day expiry ไม่ขยาย; cap access expiry; missing/expired/revoked401 |
| POST | /auth/logout | revoke represented sessionและล้างcookie; ไม่มีcookieยัง204หลังผ่านOrigin |
| GET | /me | verified account/profile/onboarding |
| GET, PUT | /me/profile | profile/preferences/derived onboarding; invalid codes rollback |
| GET | /platforms | active codes; Bearer required |
| GET, POST | /vehicles | own array0–1 / create; second vehicle409 |
| PATCH | /vehicles/{vehicle_id} | partial validated settings; nonowner404 |
| POST | /sessions/preview | read-only summary + signed15-minute preview token |
| POST | /sessions | atomic confirm/create +24h user-scoped idempotent replay |
| POST | /sessions/{session_id}/preview | merge partial edits, original snapshot + base version |
| PATCH | /sessions/{session_id} | version check, atomic nested replacement, original snapshot |
| GET | /sessions | active own history, inclusive dates, pagination/has_more |
| GET, DELETE | /sessions/{session_id} | own detail / soft delete; nonowner/deleted404 |
| GET | /reports/daily | saved snapshot summary; absent day200 data:null |

จำนวน:18 pilot operations +2 health operations ใน Swagger runtime
Swagger UI หลังเริ่มserver: `http://localhost:8080/swagger/index.html`
OpenAPI runtime: `cmd/rydo-api/swagger/openapi.json` (3.1; schema checked)
ไฟล์ proposed เดิมใน `contracts/t01/` ยังเก็บแยกไว้ ไม่ใช่หลักฐาน runtimeเก่าที่ถูกเขียนทับ

## หลักฐานและผลตรวจจริง

| เกณฑ์ | ผล | หลักฐาน |
|---|---|---|
| T03 auth/session/origin/revocation/expiry | Passed local | integration_test.go + line_test.go; external LINE transport doubleเฉพาะtest |
| T04 profile/platform/vehicle/owner | Passed local | two-account tests; unique vehicle; partial field/null validation |
| T05 input + decimal DTO | Passed local | no JSON number/negative/exponent/NaN/excess scale; summary672.50/84.06/6.73และzero rates |
| T06 preview | Passed local | no session writes; cost change/expiry/cross-user preview rejected |
| T07 atomicity/idempotency | Passed local | same key6 concurrent201→one row; distinct keys same date→one201/one409; DB trigger failure→rollback financial rows + replay row |
| T08 edit/history/report | Passed local | concurrent edits→one200/one409; historical edit after vehicle price change retains672.50 net; zero/null, pagination, inactive retained platform, soft deletion |
| Exact unknown-field rejection | Passed local | regression test rejected compound property names after red→green correction |
| HTTP wire vs accepted T01 | Passed | 143 real HTTP responses / 204 JSON Schema checks; successful coverage18/18 operations |
| Go static / formatting | Passed | go vet exit0; gofmt reports no files |
| Go tests | Passed | full-tests.log; all three packages; PostgreSQL is real/private |
| Go race | Passed | race-tests.log; all three packages, count1 |
| Build | Passed | go build -buildvcs=false ./... exit0 |
| Executable + DB foundation | Passed | foundation-http-db.log; binary exposes protected route401; migration up/down/readiness and DB ownership/NaN checks |
| Runtime OpenAPI | Passed | runtime-openapi.log;20 operations; only verified pilot operations promoted |
| Actual LINE/LIFF/browser HTTPS | Not checked | no real LINE token/account/channel action, no deploy |

Go1.25.0 linux/amd64; PostgreSQL18.6; openapi-spec-validator0.9.0 + jsonschema4.26.0
`http-capture.json` ใช้ข้อมูลสมมติและ redact ID/access/preview tokens แล้ว ไม่เก็บAuthorizationหรือcookieลงหลักฐาน
ไม่มี production bypass, dev-login route หรือ environment flag เปลี่ยน verifier
production verifierใช้ HTTPS endpointคงที่ของLINE แล้วตรวจ iss/aud/sub/exp จากผลverification
unit testแทนเฉพาะ external HTTP transport; integration testแทน verifierที่ boundary ใน `_test.go` เท่านั้น

## วิธีตรวจซ้ำ

รันใน Backend ด้วยผู้ใช้ทั่วไป มี Go/PostgreSQL server tools และ Pythonuv:

```sh
go vet ./...
RYDO_CAPTURE_HTTP="$PWD/verification/local-pilot/http-capture.json" go test -count=1 ./...
go test -race -count=1 ./...
go build -buildvcs=false ./...
uv run --with openapi-spec-validator==0.9.0 --with jsonschema==4.26.0 python scripts/validate-pilot-http.py
PATH="$HOME/go/bin:$PATH" sh scripts/test-foundation.sh
```

integration TestMain สร้าง private PostgreSQL cluster ด้วย Unix socketและปิด TCP ไม่ใช้ `DATABASE_URL` จากภายนอก
ลง migration ในclusterนั้น แล้วปิด/ลบเฉพาะไฟล์ที่testสร้าง ไม่มีการอ่านหรือเปลี่ยนข้อมูลในPostgreSQL clusterอื่นบนเครื่อง
รอบตรวจนี้ปิด API/DBชั่วคราวหลังจบ ไม่ได้เปิดserviceหรือฐานทดสอบค้างไว้ที่8080
การรันซ้ำอาจอัปเดตcapture IDs/hash ต้องเก็บหลักฐานใหม่กับrevisionเดียวกัน
ตัวตรวจT01เดิมมี historical source-hash guard จึงไม่ใช้ยืนยันruntimeใหม่; ใช้ scripts/validate-pilot-http.py และruntime schemaแทน

## การเริ่ม API สำหรับ frontend

ดู README.md และ `.env.example` ตั้ง `DATABASE_URL` ของฐานพัฒนาแยก, `WEB_APP_URL` เป็นexact browser origin,
`JWT_SIGNING_KEY`/`PREVIEW_SIGNING_KEY` เป็นค่าสุ่มคนละค่าอย่างน้อย32ไบต์ และ `LINE_LOGIN_CHANNEL_ID`
รัน migration ถึงversion3แล้ว `go run -buildvcs=false ./cmd/rydo-api`
ค่าจริงของsecretsไม่อยู่ในsource; ไม่มีdefaultkey ไม่มีfakeidentityในproduction
ถ้ายังไม่มีreal LINE Login channel/LIFF/HTTPS ให้ทดสอบได้ในtest harnessเท่านั้น ไม่เพิ่มbypassเพื่อข้ามข้อค้าง

Frontend owner (Jinbae/Rika) เชื่อมตามT01:

- เริ่มrefreshก่อน, verified `/me` เป็นaccountId; access tokenอยู่memory; same-origin proxyส่งOrigin/Set-Cookie
- readSetup=profile+vehicles; saveSetupแยกvehicleกับprofile ต้องรองรับpartial failureแล้วอ่านสถานะใหม่
- preview/editส่งเฉพาะallowedfields; PATCHไม่ส่งwork_date/vehicle_id/version/summary
- confirmคืนid/version/summaryตามสเปก แล้วGET detailเพื่อhydrate SavedDay
- retryหลังlost responseใช้body/key/previewเดิม; PREVIEW_STALE/versionชนต้องreloadและยืนยันใหม่
- listDaysต้องอ่านต่อจนhas_more=false; financial stringsไม่คำนวณใหม่ที่frontend
- schema wireผ่านไม่แทนการทดสอบadapter/LINEจริง ต้องรันE2Eร่วมอีกครั้ง

## ขอบเขตค้างก่อนข้อมูลจริง

- D01 วิธีแก้วันที่ผิด/ผู้มีสิทธิ์ยังไม่ตกลง; dateและvehicle immutable การDELETEไม่ใช่workflowแก้วันที่
- รอLIFF ID, HTTPS, real verifier/channel/originและLINE iOS/Android; ยังไม่เก็บข้อมูลไรเดอร์จริง
- T09 independent/security reviewและT10 release/privacy readinessยังไม่ปิด รวม deployment rate-limit policy
- 24h replay cutoffทำงานแล้ว แต่ background purge/retentionของexpired responsesยังเป็นงานปฏิบัติการก่อนข้อมูลจริง
- T02งานsqlc/CIที่ค้างเดิมไม่ถูกอ้างว่าเสร็จ: รอบนี้ใช้parameterized SQLผ่านpgx ไม่มี ORM/model generator หรือgo:generate/sqlc configให้รัน; ไม่เปลี่ยนschema
- ไม่มี webhook/Redis/worker ในงานนี้ และไม่อ้างว่าผ่านpilotหรือพร้อมproduction

## ไฟล์ที่เปลี่ยนและส่งต่อ

เพิ่ม internal/pilot/{server,auth,setup,workday,sessions}.go และ integration/LINE verifier tests
เพิ่ม cmd/rydo-api/pilot_test.go และ scripts/validate-pilot-http.py, scripts/build-runtime-openapi.py, .env.example
แก้ cmd/rydo-api/main.go ให้ประกอบAPIและตรวจconfig; Swagger runtime, scripts/test-foundation.shให้ทดสอบbinaryพร้อมconfig
อัปเดต README.md, Task.mdเฉพาะT03–T08; เพิ่มหลักฐานในverification/local-pilot
ไม่แก้go.mod/go.sum, profit module, migrations, frontend, PRD/API ต้นทาง

ผู้รับช่วง: Rikaตรวจdiff/หลักฐานและให้Jinbaeเชื่อมadapter ไม่ต้องทำT02foundationหรือT03–T08ซ้ำ
ตำแหน่งกลับมาทำต่อ: รายงานนี้ + source-manifest.json + Task.md T03–T08
