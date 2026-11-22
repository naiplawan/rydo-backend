# T01 — Proposed pilot integration contract

สถานะ: **Partial — สัญญาที่เสนอสำหรับ implementation review; ยังไม่ใช่ runtime API**
ผู้รับผิดชอบ: Ferris ตามงานจาก Rika; วันที่เริ่ม 21 พฤศจิกายน 2026 (Asia/Bangkok)

- [OpenAPI 3.1 JSON](openapi.proposed.json): 13 paths / 18 operations ใต้ `/api/v1`
- [ตัวอย่างและกรณีขอบเขต](examples.json): ข้อมูลสมมติ ไม่มี token/บัญชีจริง
- [ตัวตรวจ](validate_contract.py): OpenAPI structure, local refs, positive/negative examples และสูตรตัวอย่างอิสระ
- [ผลตรวจจริง](VERIFICATION.md) และ [source hashes](source-baseline.json)
- [Task T01](../../Task.md#t01), [SRS R02/R06/R16](../../SRS.md), [PRD](../../../rydo-doc/01_PRD.md)

## ขอบเขตและการแยก runtime

อ่าน Task T01, แผนขั้น pilot, [API](../../../rydo-doc/04_API_SPECIFICATION.md), migrations 000001/000002 และตรวจ 000003 ที่เพิ่มระหว่างทำงาน,
[frontend contract](../../../rydo-frontend/docs/04_API_CONTRACT.md) และ [TrialData](../../../rydo-frontend/src/trial-data.ts) ก่อนจัดทำ
ยืนยัน ownership จากคำมอบหมาย Rika และไม่พบ T01 artifact/lock เดิม จากนั้นสร้าง directory นี้แบบ exclusive
ไม่มีการแก้ frontend, runtime Go, migration, PRD/API ต้นทาง หรือ T02–T08

`openapi.proposed.json` มี `x-contract-status: proposed` และ `x-runtime-implemented: false` ทุก operation
ไม่มี health/webhook/notification-preference operations ในสัญญาชุดนี้
ห้ามนำไปแทนไฟล์ `cmd/rydo-api/swagger/openapi.json` หรือโฆษณาว่ามี endpoint แล้ว
ไฟล์ Swagger runtime เดิมมีเพียง `/health/live` และ `/health/ready`; คง hash เดิมไว้ใน source baseline
เมื่อ implement endpoint จริง เจ้าของ runtime ต้องเพิ่มเฉพาะ operation ที่ผ่านการทดสอบแล้วใน Swagger ของ runtime
ชื่อ `servers: /` ใน proposed หมายถึง same-origin proxy ที่จะสร้าง ไม่ใช่ URL ที่ยืนยันว่าพร้อมใช้

## ข้อตกลงที่ยึดตามสเปกต้นทาง

- POST auth/line, refresh, logout ตรวจ Origin ตรง web origin แบบ exact ก่อนดำเนินการ
- login ตรวจ LINE ID token ฝั่งเซิร์ฟเวอร์ ไม่ใช้ fixture accountId หรือ client LINE user ID เป็นตัวตน
- access token อายุสูงสุด 900 วินาทีอยู่ใน memory; session 30 วันไม่ต่ออายุเมื่อ refresh
- refresh cookie `__Host-rydo_refresh`: `Path=/; Secure; HttpOnly; SameSite=Lax` ไม่มี Domain; DB เก็บ hash เท่านั้น
  proxy ต้องส่งต่อ Origin/Set-Cookie; logout ไม่มี cookie ยังได้ 204 หลังผ่าน Origin check
- ทุก protected request ตรวจ session expiry/revocation และ owner; nonowned/deleted resource คืน 404
  ข้อมูลส่วนตัวและ auth ใช้ `Cache-Control: no-store`; ไม่มี raw LINE subject, token หรือ financial input ใน log
- เงิน/ระยะ/ชั่วโมงเป็น decimal strings ห้ามใช้ JSON number; ชั่วโมงส่งเป็นทศนิยม ไม่ใช่ HH:mm
  รูป precision/scale อิง migration กับ profit code; summary ใช้ชื่อ field ตาม API §7 ไม่ใช่ชื่อ exported Go struct
- ปัด fuel และ reserve แยกกัน 2 ตำแหน่ง half away from zero แล้วรวมต้นทุน/หักกำไร ค่อยหารเป็น rate
  ไม่ปัด distance/efficiency กลางทาง; ศูนย์ให้ rate เป็น null; normalize negative zero
- preview ไม่เขียน session/entries และไม่ต้องมี idempotency key; signed token อายุ 15 นาทีผูก user, target,
  normalized input, cost snapshot และ base version ของ edit
- create snapshot จากค่ารถปัจจุบัน ต้อง lock/recheck ค่ารถใน transaction ตอนยืนยัน
  edit/report ใช้ snapshot ที่บันทึกกับรายการเดิมเสมอ แม้ค่ารถเปลี่ยนแล้ว
- confirm ใช้ `POST /sessions` หรือ `PATCH /sessions/{session_id}` ไม่มี endpoint `/confirm` ใหม่
  header บังคับ `X-Preview-Token` + `Idempotency-Key` (UUID)
- รายการและ nested entries กับผล replay ต้อง commit/rollback พร้อมกัน; หนึ่ง active record ต่อบัญชีต่อวัน Bangkok
  PATCH increment version แบบ atomic และไม่รับ date/vehicle/snapshot จาก client
- read/history คืน saved fields + id/version/summary; daily report ไม่มีรายการได้ 200 `data:null`

## ค่าที่เสนอชั่วคราวและย้อนเปลี่ยนได้

ตัวเลือก P01–P08 ต่อไปนี้เติมช่องว่างจาก frontend M-03..M-19 เพื่อให้ทำ typed adapter และ contract tests ได้
เป็น local proposed baseline สำหรับ Rika/เจ้าของ API–Frontend review ไม่ใช่การแก้สเปกผลิตภัณฑ์หรือยืนยัน runtime
หากจะเปลี่ยนหลังมี adapter ให้แก้ schema/examples/tests ร่วมกันก่อนส่งมอบ

| ID | ข้อเสนอ | ผลต่อ integration / ที่มา |
|---|---|---|
| P01 | note ≤1000 Unicode code points, expenses ≤100 แถว, income ≤4 รหัสแพลตฟอร์ม, decimal input ≤32 ตัวอักษร, ID token ≤16384; page_size สูงสุด100, page สูงสุด2147483647; version ไม่เกิน JSON safe integer | ข้อจำกัดป้องกัน payload ใหญ่แบบ reversible; ยังไม่ใช่ค่าที่พิสูจน์จาก pilot; precision/scale ยังอิง DB; output totals ไม่จำกัดตามคอลัมน์ input รายแถว |
| P02 | `/me` และ profile คืน Profile; vehicles GET คืน array0–1; mutations คืน object; history meta `{page,page_size,has_more}`; from/to ไม่ส่งคือไม่จำกัดขอบนั้น; platforms ต้อง Bearer | response ที่เดิมไม่กำหนด; ไม่มี total ที่ต้องคำนวณ; order `work_date DESC,id DESC`; defaults page1/page_size20; ชื่อ note ที่ DB null แปลงเป็น empty string |
| P03 | `display_name` ใน User/Profile nullable; explicit PUT รับ string ≤120; ไม่เรียก LINE profile เพิ่มหรือเปลี่ยน scopes | openid-only อาจไม่มีชื่อ ไม่แต่งชื่อ; client ห้ามพึ่งชื่อเพื่อระบุตัวตน |
| P04 | vehicle PATCH partial; session PATCH partialตามสเปก; null/empty object/unknown fields ไม่รับ; noteว่างล้าง, expense[]ล้าง, income[]ไม่รับ | Omitted retains; arrays replace. Request fingerprint normalize object keys/UUID/decimal แต่รักษาชุด field ที่ส่งกับลำดับ arrays; create เติม optional expenses/note defaults ก่อน hash |
| P05 | error.details เป็น array ของ validation `{field,code,message}`, own-conflict `{session_id}` หรือ `{reason}`; เพิ่ม generic error codes ตาม schema | คงห้าชื่อเดิม; versionชนใช้409 PREVIEW_STALE + VERSION_CHANGED และต้อง reload ก่อน previewใหม่; auth401, Origin403, malformed400, missing/invalid confirm header422; 429 มี Retry-After seconds |
| P06 | คง create/PATCH response ขั้นต่ำ `{data:{id,version,summary}}`; frontend GET detail หลังสำเร็จ | ไม่เปลี่ยน API ต้นทางให้คืน SavedDay ทั้งก้อนตามที่ fixture คาด; ถ้า GET ล้มเหลวต้องแยก “บันทึกสำเร็จแต่โหลดรายละเอียดไม่ได้” จาก “ยังไม่รู้ว่าบันทึกสำเร็จ” |
| P07 | ใช้ calendar date RFC3339 full-date ช่วงปี0001..9999 และห้ามเกินวันนี้ Bangkok; ยังไม่ตั้ง earliest work date เชิงธุรกิจ | เป็นขอบเขต serialization เพื่อให้ JSON/OpenAPI ตรวจได้ ไม่ห้ามวันย้อนหลังตามสมมติฐานธุรกิจใหม่ |
| P08 | ลำดับ conflict: authenticate/header syntax → serialize key/replay → ownership → date conflict (create) → preview validity/version | เมื่อมีทั้งวันที่ซ้ำกับ previewหมดอายุ ให้ SESSION_DATE_EXISTS เพื่อเปิดรายการเดิม; preview endpointตรวจข้อมูล/ownershipแต่ไม่กันวันที่ซ้ำ ใช้ createเป็น final guard |

### Canonical request และ replay

P04: ตรวจ input ก่อน normalize; decimalที่ถูกต้องแปลงเป็นค่าฐานสิบ canonical (ตัด leading/trailing zeros, ศูนย์เป็น `0`)
ไม่ trim/เปลี่ยน note และไม่ยอมรับ comma/exponent/sign เพราะ normalization ไม่ใช่การแก้ input ที่ผิด
UUID แปลงเป็นรูปแบบ canonical lowercase; object keys เรียง, arrays รักษาลำดับ
create เติม default expenses=[], note="" และ expense.note="" เมื่อไม่ส่ง
PATCH รักษา field-set ที่ส่ง (omitted ต่างจาก supplied) แล้ว bind ทั้ง patch และค่าที่ merge แล้วกับ token

fingerprint รวม method ตัวใหญ่, concrete path ที่ normalize UUID, canonical request และ **preview token เดิมทั้งค่า**
ไม่รวม access token หรือ idempotency key ใน fingerprint; scope stored key ด้วย verified user
หลังตรวจ auth/header syntax แล้ว matching successful replay ภายใน24ชั่วโมงต้องมาก่อน date/version/preview expiry check
ไม่ให้ tokenที่หมดอายุทำให้ lost-response retry ที่เคย commitแล้วล้มเหลว
keyเดิมแต่ fingerprintต่างได้409; failed writeไม่เก็บผลสำเร็จ; หลัง24ชั่วโมง clientต้องโหลดรายการปัจจุบันก่อนส่งใหม่
logout/revocationยังต้องปฏิเสธ protected requests แม้มี replay row

## ประเด็นตัดสินจริงที่ยังไม่ปิด

| ID | ต้องตัดสินอะไร / ใคร | ขวางอะไร | งานที่เดินต่อได้ |
|---|---|---|---|
| D01 | Product + เจ้าของ API + ผู้ใช้: วิธีช่วยแก้วันที่ผิดระหว่าง pilot ใครมีสิทธิ์ มีคำขอจากเจ้าของอย่างไร และหลักฐานการแก้ | T01 ยัง Partial สำหรับ correction workflow และ T10 ก่อนใช้ข้อมูลจริง; สเปกยัง immutable date/vehicle จึงไม่เพิ่ม endpoint เอง; DELETE/recreateไม่ใช่วิธีแก้วันที่ที่ปลอดภัย | T03–T08 สำหรับ create/read/editค่าที่อนุญาต; frontendคงวันที่ล็อกหลังบันทึกและตรวจให้ชัดก่อนยืนยัน |
| D02 | ผู้ดูแล LINE/Infra: LIFF ID, HTTPS, exact web origin และ LINE Login/Messaging อยู่ provider เดียวกัน | ทดสอบ loginจริง/LINE E2E; ไม่แก้บัญชีในงานนี้ | พัฒนา authกับ verifierจำลองใน test, contract adapter และ proxy tests ด้วยค่าทดสอบ; ต้องไม่มี bypass ใน production |

คำอธิบายชั่วโมงรวมใช้ข้อความชั่วคราว “รวมเวลารอรับงาน หักเวลาพักส่วนตัว เปิดหลายแอปพร้อมกันให้นับเวลาครั้งเดียว”
ตามงานวิจัย/ชุดส่งมอบดีไซน์ แล้วตรวจความเข้าใจกับไรเดอร์; APIรับเพียง working_hoursรวม ไม่เพิ่มฟิลด์เวลาใหม่
ค่ารถ40/0.600เป็นข้อมูลตัวอย่างจาก API ไม่ใช่ presetที่พิสูจน์แล้ว ก่อนใช้จริง Productต้องยืนยันแหล่งหรือให้กรอกเอง

## การเทียบกับ migration และโค้ด

- 000001 มี users/auth_sessions/preferences/vehicles/daily_sessions/income/expenses/session_requests และ indexes ที่จำเป็น
  ไม่ต้องสร้างตารางใหม่เพื่อรูป response ที่เสนอ
- 000003 ที่เพิ่มระหว่างตรวจห้าม NaN ในต้นทุน ระยะทาง และยอดเงิน; รูป decimal ที่เสนอกัน NaN อยู่แล้ว จึงไม่เปลี่ยน wire contract
- 000002 ผูก `(vehicle_id,user_id)` กับ vehicle owner: ช่วยป้องกัน cross-user association แต่ handler ยังต้องกรอง ownerทุก query
- ไม่ส่ง snapshot ใน request และไม่เพิ่ม public snapshot mutation; version cap P01 อยู่ที่ API boundary ก่อน BIGINT เกิน safe integer
- ยังต้อง implement DTO สำหรับ `profit.Summary` ซึ่งไม่มี JSON tags และไม่มี date/currency/input metrics ตาม wire contract
- การอนุญาต platform inactiveเดิมอยู่ใน service validation ไม่ใช่ JSON Schema; `uniqueItems` อย่างเดียวกัน platformซ้ำต่างยอดไม่ได้
- Schemaตรวจรูปแบบได้ แต่ future date, owner, token signature, locks, atomicity, expiry/replay และ DB effects ต้องมี integration tests จริง

## ลำดับ Backend และ Frontend หลัง review T01

1. Rika/เจ้าของสัญญาตรวจ P01–P08 และรับ baseline; บันทึก D01 พร้อมผู้ตัดสิน ไม่อ้างว่า correction พร้อม
2. ตรวจของที่มีอยู่จาก T02 ก่อน: health/migration/profit code ไม่ทำซ้ำ; เติมเฉพาะสิ่งที่ขาดตามเจ้าของ implementation
3. Backend T03 auth/session → T04 profile/platform/vehicle; test invalid token, Origin, expiry/revocation และสองบัญชี
4. Backend T05 validators+DTO → T06 preview → T07 transaction/idempotency → T08 history/detail/edit/report
   ทุกขั้นตรวจ against contract; preview/create/editใช้สูตรเดียว; ขอบเขต header, null, lists และ conflicts ตาม schema+คำอธิบาย
5. Frontend ทำ typed HTTP adapter/fixtures/field-error mapper ตาม proposedได้ทันทีหลังรับ baseline โดยยังไม่เปลี่ยน modeเป็นlive
   `TrialData.accountId` ต้องมาจาก verified `/me`, ไม่ใช่ fixture ID; refreshได้ครั้งเดียวและใช้ payload/key/tokenเดิมบน retry
6. `readSetup` เรียก profile+vehicles (ไม่มีรถคือยังไม่ตั้งค่า); `saveSetup` สร้าง/แก้รถแล้ว PUT profile
   สอง endpointนี้ไม่ใช่ transactionเดียว ต้องรองรับรถสำเร็จแต่ profileล้มเหลวด้วยการอ่านสถานะใหม่ ไม่อ้าง onboardingเสร็จ
7. `readDay` ใช้ from=to แล้ว detailถ้าจำเป็น; `listDays` ต้องตาม has_more; preview edit และ confirm editส่งเฉพาะ SessionPatch
   **ห้ามส่ง work_date/vehicle_id/version/summary ทั้ง SavedDay กลับเข้า PATCH** แม้ fixtureรับ WorkdayInputทั้งก้อน
   confirm คืนผลขั้นต่ำแล้ว GET detail; แยก error ของ follow-up GET จากความไม่แน่นอนของ save
8. เมื่อ endpointsมีจริงจึงสลับ live adapterและรัน smoke/API/E2E ใหม่ รวม lost response/stale preview/versionชน/เปลี่ยนรถแต่ snapshotเดิม
   fixture30/30ไม่ใช่หลักฐานของ Backend runtime; LINE E2Eต้องรอ D02 และสิทธิ์เปิดใช้แยก

## วิธีตรวจซ้ำ

```sh
uv run --with openapi-spec-validator==0.9.0 --with jsonschema==4.26.0 python contracts/t01/validate_contract.py
```

รันจาก Backend; validatorไม่เปิด serviceหรือเชื่อมLINE/DB เก็บผลจริงและเวอร์ชันเครื่องมือใน VERIFICATION.md
ทดสอบฝั่ง Backendเดิมด้วย Go vet/test/build เป็น baselineเท่านั้น ไม่อ้างว่าทดสอบ endpointsที่ยังไม่มี

## การเปลี่ยน source ระหว่างงาน

ตัวตรวจเคยหยุดเมื่อ main.go เปลี่ยน requiredSchemaVersion จาก2เป็น3 โดยงานอื่น พร้อม migration000003 กัน NaN
อ่านไฟล์ใหม่แล้วสัญญาไม่เปลี่ยน จึงเก็บ hashแรกใน source-baseline.json และ hashหลังทบทวนใน [source-reviewed.json](source-reviewed.json)
ตัวตรวจเทียบกับ hashหลังทบทวนและจะหยุดอีกหาก source เปลี่ยน ไม่อ้างว่า Ferris เป็นผู้แก้ foundation
