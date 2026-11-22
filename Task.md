# Backend Task register

วันที่จัดทำ: 21 พฤศจิกายน 2026 | สถานะ: เริ่มงานฐานระบบและสูตรคำนวณแล้ว

## วิธีอ่านและใช้

เส้นทางตรวจย้อนกลับ: Task → รหัสใน [SRS ฉบับร่าง](SRS.md) → [PRD v1.1](../rydo-doc/01_PRD.md) และ API/Schema/Architecture ต้นทาง
แต่ละ Task มีลิงก์ PRD โดยตรงด้วย รหัส T00–T17 เป็นรหัสภายในไฟล์นี้; RP-xxx อ้าง [backlog เดิม](../rydo-doc/05_IMPLEMENTATION_BACKLOG.md)
T02 และ T05 มีงานบางส่วนพร้อมผลตรวจ สถานะของงานอื่นให้ดูในรายการแต่ละงาน
Owner ที่ระบุเป็นบทบาทที่ต้องยืนยัน ไม่ใช่การจ่ายงานให้เอเจนต์อื่น

- สถานะเริ่มต้น: Proposed / Not started; ถ้า dependency หรือการตัดสินใจยังขาด ให้บันทึก Blocked พร้อมสาเหตุ
- ทำเครื่องหมายเสร็จได้เมื่อเกณฑ์รับงานผ่าน มี revision/ผลทดสอบ และเจ้าของรับงาน; หากใช้บางส่วนไม่ได้ให้ระบุ Not checked
- งานต้นแบบใช้เส้นทางขั้นต่ำและมีประตู T00; งาน MVP ที่เลื่อนออกไปต้องผ่าน T12 ก่อน
- รายละเอียดเหตุผลอยู่ใน [แผน Backend](IMPLEMENTATION_PLAN.md) และ [ผลตรวจ](BACKEND_REVIEW.md)

## รายการงาน

<a id="t00"></a>
### [ ] T00 — ยืนยันขอบเขตและเจ้าของต้นแบบ

- ระยะ: ก่อนต้นแบบ | สถานะ: Proposed / Not started
- Owner: ผู้ใช้/เจ้าของผลิตภัณฑ์และผู้ประสานงาน
- ขึ้นต่อ: ไม่มีงานก่อนหน้า; ต้องมีการตัดสินใจในงานนี้ก่อนเริ่มสร้าง
- SRS: [R01](SRS.md#r01), [R16](SRS.md#r16)
- PRD: [§11](../rydo-doc/01_PRD.md#11-success-metrics), [§7](../rydo-doc/01_PRD.md#7-core-user-journeys)
- Backlog เดิม: งานเตรียมเพิ่มเติมจากการตรวจธุรกิจ ไม่ใช่ RP ใหม่
- เกณฑ์รับงาน: มีบันทึกผู้ใช้ยืนยันขอบเขต เครื่องมือ งบเวลา เจ้าของ FE/BE/LINE และวิธีแก้ข้อมูลผิด; ยังไม่อนุมัติ MVP เต็ม
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ตรวจบันทึกตัดสินใจและผู้รับผิดชอบ ไม่ใช้คำแนะนำเอเจนต์แทนการอนุมัติ
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t01"></a>
### [ ] T01 — ปิดสัญญาและช่องว่างก่อนลงมือ

- ระยะ: ก่อนต้นแบบ | สถานะ: Partial — proposed pilot contract validated; D01/date-correction workflow ยังเปิด
- Owner: Ferris สำหรับ artifact T01; เจ้าของสัญญา/ผลิตภัณฑ์รับคำตัดสิน D01 และ review P01–P08
- ขึ้นต่อ: [T00](#t00)
- SRS: [R02](SRS.md#r02), [R06](SRS.md#r06), [R16](SRS.md#r16)
- PRD: [§5](../rydo-doc/01_PRD.md#5-product-goals), [§9](../rydo-doc/01_PRD.md#9-business-rules), [§7](../rydo-doc/01_PRD.md#7-core-user-journeys)
- Backlog เดิม: RP-006; RP-023; RP-032
- เกณฑ์รับงาน: OpenAPI ครอบคลุมเส้นทางต้นแบบ; ระบุ input/array/note/page limits และ error; ตกลงคำอธิบายชั่วโมงและวิธีช่วยแก้วันที่พร้อมสิทธิ์; รายการใหม่ที่ยังไม่ตกลงมีสถานะเปิด
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ตรวจ schema และเทียบ PRD/API; วันที่ปลายทางซ้ำต้องไม่ทับ; ไม่เปลี่ยน snapshot จาก client โดยไม่มีสิทธิ์
- ผลจริง: OpenAPI 3.1 สำหรับ 18 operations ผ่าน structure/schema/examples; ยังไม่ implement API และไม่อ้าง runtime พร้อม
- Artifact: [OpenAPI proposed](contracts/t01/openapi.proposed.json), [ข้อตกลง/ข้อค้างและลำดับ integration](contracts/t01/README.md), [หลักฐานตรวจ](contracts/t01/VERIFICATION.md)
- คำตัดสินค้าง: D01 วิธีช่วยแก้วันที่และผู้มีสิทธิ์; งานอิสระ T03–T08 เดินตาม proposed baseline ได้หลัง review ไม่ต้องรอเปลี่ยนบัญชี LINE

<a id="t02"></a>
### [ ] T02 — ฐานระบบและ persistence

- ระยะ: ต้นแบบ | สถานะ: In progress
- Owner: Backend; ต้องยืนยันผู้รับผิดชอบก่อนเริ่ม
- ขึ้นต่อ: [T01](#t01)
- SRS: [R02](SRS.md#r02), [R14](SRS.md#r14)
- PRD: [§5](../rydo-doc/01_PRD.md#5-product-goals), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-002–007; RP-047 (ส่วนจำเป็น)
- เกณฑ์รับงาน: server Go/Gin เริ่มได้; PostgreSQL migration up/down และ sqlc compile; health/log/config ไม่มี secret; CI มี static/build/test
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ใช้ DB ทดสอบรัน migration/rollback และ build/generated queries; บันทึกคำสั่งและผลจริง
- ผลจริง: Go/Gin health และ Swagger ของ health endpoints เปิดได้ ทดสอบ PostgreSQL 18 สำหรับ migration up/down, readiness, เจ้าของรถและค่าตัวเลขผ่าน `go vet ./...`, `go test -race -count=1 ./...` และ `go build -buildvcs=false ./...` ผ่าน
- CI/CD: มี workflow สำหรับ static analysis, tests, build และ Railway pilot แล้ว รอผลรันบน GitHub และ Railway จริง
- งานคงเหลือ: sqlc และตรวจผล CI/CD บน GitHub/Railway
- Artifact: [API](cmd/rydo-api/main.go), [OpenAPI](cmd/rydo-api/swagger/openapi.json), [migration ล่าสุด](db/migrations/000003_finite_workday_numbers.up.sql), [integration test](scripts/test-foundation.sh), [CI/CD](.github/workflows/ci-cd.yml), [การตั้งค่า Railway](DEPLOYMENT.md)

<a id="t03"></a>
### [ ] T03 — LINE login และ session

- ระยะ: ต้นแบบ | สถานะ: Implemented / local checks passed — รอ Rika review; ยังไม่ผ่าน LINE จริง
- Owner: Ferris (Backend ตามงานจาก Rika)
- ขึ้นต่อ: [T02](#t02)
- SRS: [R03](SRS.md#r03), [R11](SRS.md#r11)
- PRD: [§7](../rydo-doc/01_PRD.md#7-core-user-journeys), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-010–012
- เกณฑ์รับงาน: verified subject สร้าง/โหลด user; cookie/access token ตามสเปก; refresh/logout ตรวจ Origin; session หมดอายุหรือ revoke ใช้ token เดิมไม่ได้
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ทดสอบ token ผิด audience/expiry, Origin ผิด, logout, session expiry และสองบัญชี
- ผลจริง: ผ่าน Go vet/test/race/build และ HTTP กับ PostgreSQL แยก; หลักฐานตาม [รายงาน local pilot](verification/local-pilot/REPORT.md)
- Artifact: `internal/pilot/` และ Swagger runtime; ไม่แก้ frontend/LINE; D01 คงวันที่ immutable

<a id="t04"></a>
### [ ] T04 — Profile แพลตฟอร์มและรถ

- ระยะ: ต้นแบบ | สถานะ: Implemented / local checks passed — รอ Rika review; ยังไม่ผ่าน LINE จริง
- Owner: Ferris (Backend ตามงานจาก Rika)
- ขึ้นต่อ: [T03](#t03)
- SRS: [R04](SRS.md#r04), [R05](SRS.md#r05), [R11](SRS.md#r11)
- PRD: [§8](../rydo-doc/01_PRD.md#8-functional-requirements), [§7](../rydo-doc/01_PRD.md#7-core-user-journeys), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-013–019 (ส่วน API)
- เกณฑ์รับงาน: profile/preferences และหนึ่งรถต่อคนทำงาน; ค่าไม่ถูกต้องถูกปฏิเสธ; รถคันที่สองได้ 409; เปลี่ยนค่ารถไม่เปลี่ยนรายการเก่า; ค่าแนะนำถ้ามีต้องมีแหล่งที่มา
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ทดสอบ CRUD ตาม endpoints ที่สเปกมี, validation, active codes และ ownership; ตรวจ preset กับเจ้าของผลิตภัณฑ์
- ผลจริง: ผ่าน Go vet/test/race/build และ HTTP กับ PostgreSQL แยก; หลักฐานตาม [รายงาน local pilot](verification/local-pilot/REPORT.md)
- Artifact: `internal/pilot/` และ Swagger runtime; ไม่แก้ frontend/LINE; D01 คงวันที่ immutable

<a id="t05"></a>
### [ ] T05 — Validator และสูตร decimal

- ระยะ: ต้นแบบ | สถานะ: In progress
- Owner: Ferris (Backend ตามงานจาก Rika)
- ขึ้นต่อ: [T01](#t01), [T02](#t02)
- SRS: [R06](SRS.md#r06), [R07](SRS.md#r07)
- PRD: [§9](../rydo-doc/01_PRD.md#9-business-rules), [§9.6](../rydo-doc/01_PRD.md#96-metrics)
- Backlog เดิม: RP-021–023; RP-027–031; RP-043
- เกณฑ์รับงาน: รับ decimal strings ตาม precision/scale; ไม่รับ fuel expense; ผลตัวอย่างได้ 672.50, 84.06, 6.73; ties ทั้งสองทิศถูกต้อง; ตัวหารศูนย์ได้ null
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: unit tests สำหรับค่าลบ ขอบเขต/scale แพลตฟอร์มซ้ำ ขาดทุน ศูนย์ และการไม่ปัดค่ากลางก่อนเวลา
- ผลจริง: มีสูตร decimal และ unit tests สำหรับตัวอย่างหลัก การปัดเศษ ขาดทุน ตัวหารศูนย์ และค่า distance ผิดรูปแบบหรือเกิน precision
- งานคงเหลือ: validation ระดับ API เช่น หมวดค่าใช้จ่ายและแพลตฟอร์มซ้ำ รวมถึงนำสูตรไปใช้ใน preview, save และ report
- Artifact: [สูตรคำนวณ](internal/profit/profit.go), [unit tests](internal/profit/profit_test.go)

<a id="t06"></a>
### [ ] T06 — Preview ที่ไม่เขียนข้อมูล

- ระยะ: ต้นแบบ | สถานะ: Implemented / local checks passed — รอ Rika review; ยังไม่ผ่าน LINE จริง
- Owner: Ferris (Backend ตามงานจาก Rika)
- ขึ้นต่อ: [T04](#t04), [T05](#t05)
- SRS: [R08](SRS.md#r08), [R11](SRS.md#r11)
- PRD: [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-026–027; RP-032 (ส่วน preview)
- เกณฑ์รับงาน: token ผูกบัญชี input snapshot target/version ตาม API; preview ไม่สร้าง session หรือ entries; สรุปใช้สูตรเดียวกับ save
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: เทียบจำนวนข้อมูลก่อน/หลัง; ทดสอบ token หมดอายุ/ปลอม/ข้ามบัญชีและค่า input เปลี่ยน
- ผลจริง: ผ่าน Go vet/test/race/build และ HTTP กับ PostgreSQL แยก; หลักฐานตาม [รายงาน local pilot](verification/local-pilot/REPORT.md)
- Artifact: `internal/pilot/` และ Swagger runtime; ไม่แก้ frontend/LINE; D01 คงวันที่ immutable

<a id="t07"></a>
### [ ] T07 — ยืนยันบันทึก atomic และ idempotent

- ระยะ: ต้นแบบ | สถานะ: Implemented / local checks passed — รอ Rika review; ยังไม่ผ่าน LINE จริง
- Owner: Ferris (Backend ตามงานจาก Rika)
- ขึ้นต่อ: [T06](#t06)
- SRS: [R06](SRS.md#r06), [R08](SRS.md#r08), [R09](SRS.md#r09)
- PRD: [§9](../rydo-doc/01_PRD.md#9-business-rules), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-020–025; RP-032; RP-044 (ส่วน save)
- เกณฑ์รับงาน: บันทึกหลักและลูกพร้อมผล replay ใน transaction เดียว; วันที่ซ้ำได้ 409; key เดิมคำขอเดิมได้ผลเดิมแม้ preview หมดอายุ; key เดิมต่างคำขอ conflict
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: integration test lost response, same-key concurrency, different-key same-date concurrency, rollback และ PREVIEW_STALE; ใช้ DB จริงสำหรับทดสอบ
- ผลจริง: ผ่าน Go vet/test/race/build และ HTTP กับ PostgreSQL แยก; หลักฐานตาม [รายงาน local pilot](verification/local-pilot/REPORT.md)
- Artifact: `internal/pilot/` และ Swagger runtime; ไม่แก้ frontend/LINE; D01 คงวันที่ immutable

<a id="t08"></a>
### [ ] T08 — แก้รายการ ประวัติ และ daily report

- ระยะ: ต้นแบบ | สถานะ: Implemented / local checks passed — รอ Rika review; ยังไม่ผ่าน LINE จริง
- Owner: Ferris (Backend ตามงานจาก Rika)
- ขึ้นต่อ: [T07](#t07)
- SRS: [R10](SRS.md#r10), [R11](SRS.md#r11)
- PRD: [§9](../rydo-doc/01_PRD.md#9-business-rules), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-020; RP-032–034
- เกณฑ์รับงาน: GET/PATCH/DELETE ที่เป็นของตนทำงาน; history/report ใช้ snapshot เดิม; PATCH ตรวจ version และแทน arrays ตามสเปก; soft delete ไม่รวมรายงาน; วันว่าง data:null
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: เปลี่ยนค่ารถแล้วอ่าน/แก้รายการเก่า; ทดสอบ version ชน อัตรา null pagination/date filters และข้ามบัญชี; การย้ายวันที่แยก T13
- ผลจริง: ผ่าน Go vet/test/race/build และ HTTP กับ PostgreSQL แยก; หลักฐานตาม [รายงาน local pilot](verification/local-pilot/REPORT.md)
- Artifact: `internal/pilot/` และ Swagger runtime; ไม่แก้ frontend/LINE; D01 คงวันที่ immutable

<a id="t09"></a>
### [ ] T09 — ตรวจสิทธิ์และข้อมูลอ่อนไหว

- ระยะ: ก่อนใช้ข้อมูลจริง | สถานะ: Proposed / Not started
- Owner: Backend; ต้องยืนยันผู้รับผิดชอบก่อนเริ่ม
- ขึ้นต่อ: [T03](#t03), [T08](#t08)
- SRS: [R11](SRS.md#r11), [R14](SRS.md#r14)
- PRD: [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-019; RP-046
- เกณฑ์รับงาน: บัญชี A ใช้ id ของ B แล้วอ่าน/แก้ไม่ได้ทุก protected endpoint; private responses ไม่เข้า shared cache; log/analytics ไม่รั่ว token/ข้อมูลการเงิน
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: negative integration tests สองบัญชี; ตรวจ headers และ log จากข้อมูลทดสอบ; บันทึกผลตรวจ security
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t10"></a>
### [ ] T10 — เชื่อมต้นแบบและตรวจพร้อมทดลอง

- ระยะ: ก่อนใช้ข้อมูลจริง | สถานะ: Proposed / Not started
- Owner: Backend ร่วมกับ FE/LINE และผู้ตรวจรับ
- ขึ้นต่อ: [T08](#t08), [T09](#t09)
- SRS: [R01](SRS.md#r01), [R14](SRS.md#r14), [R15](SRS.md#r15)
- PRD: [§11](../rydo-doc/01_PRD.md#11-success-metrics), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria), [§10](../rydo-doc/01_PRD.md#10-ux-requirements)
- Backlog เดิม: RP-008–009; RP-018; RP-026; RP-045 (ส่วนเส้นทางต้นแบบ)
- เกณฑ์รับงาน: ร่วมกับเจ้าของ FE/LINE ตรวจ onboarding→preview→ยืนยัน→ประวัติบน LINE iOS/Android; HTTPS/privacy/backup-restore ขั้นจำเป็นพร้อม; retry ไม่ซ้ำ; มีวิธีช่วยแก้วันที่ที่รักษาต้นทุนเดิม
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: E2E บน LINE จริงและสองบัญชี; ทดสอบเน็ตหลุด; ยังไม่เริ่มเก็บข้อมูลจริงถ้าหลักฐานความพร้อมขาด; ไม่ต้องรอ T14/T15
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t11"></a>
### [ ] T11 — ทดลอง 7 วันและเก็บหลักฐานธุรกิจ

- ระยะ: ทดลอง | สถานะ: Proposed / Not started
- Owner: ผู้ดำเนินการทดลองร่วมกับธุรกิจ
- ขึ้นต่อ: [T10](#t10)
- SRS: [R01](SRS.md#r01), [R15](SRS.md#r15)
- PRD: [§11](../rydo-doc/01_PRD.md#11-success-metrics), [§10](../rydo-doc/01_PRD.md#10-ux-requirements)
- Backlog เดิม: ส่วนทดลองก่อน MVP; RP-049 ใช้เป็นแนววัดผล ไม่ใช่รับรอง RP-048 เสร็จ
- เกณฑ์รับงาน: ผู้ดำเนินการที่ได้รับมอบหมายเก็บผล 5 คนจากอย่างน้อยสองกลุ่ม ใช้ ≥2 แอปและคาดวิ่ง ≥3 วัน; ให้ลิงก์/คำแนะนำครั้งแรกแล้วไม่เตือนรายวัน
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: รายงานจำนวนกรอกเอง เข้าใจต้นทุน เวลา end-to-end การกลับใช้เอง และการเปลี่ยนวัน/ชั่วโมงจริง; แยกการช่วยเหลือและความตั้งใจออกจากพฤติกรรมจริง
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t12"></a>
### [ ] T12 — ตัดสินใจลงทุน MVP

- ระยะ: ประตู MVP | สถานะ: Proposed / Not started
- Owner: ผู้ใช้/เจ้าของผลิตภัณฑ์และผู้ประสานงาน
- ขึ้นต่อ: [T11](#t11)
- SRS: [R01](SRS.md#r01)
- PRD: [§11](../rydo-doc/01_PRD.md#11-success-metrics)
- Backlog เดิม: เกณฑ์ธุรกิจ §3/§7 ไม่ใช่ RP ใหม่
- เกณฑ์รับงาน: เทียบ 4/5 กรอกครั้งแรกเอง, 4/5 เข้าใจต้นทุน, มัธยฐาน ≤30 วินาที, 3/5 บันทึกเอง ≥3 วัน, 2/5 เปลี่ยนการตัดสินใจจริง; ผู้ใช้บันทึกไปต่อ/ปรับ/ทดลองซ้ำ
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ตรวจผลทดลองและการตัดสินใจจากผู้ใช้; เกณฑ์ผ่านไม่อนุมัติอัตโนมัติ; ราคา 49 บาททดสอบแยกหลังใช้ซ้ำ
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t13"></a>
### [ ] T13 — ปิดการแก้วันที่สำหรับ MVP

- ระยะ: MVP หลังอนุมัติ | สถานะ: Proposed / Not started
- Owner: เจ้าของสัญญาร่วมกับ Backend/ผลิตภัณฑ์
- ขึ้นต่อ: [T12](#t12), [T01](#t01), [T08](#t08)
- SRS: [R10](SRS.md#r10), [R16](SRS.md#r16)
- PRD: [§9](../rydo-doc/01_PRD.md#9-business-rules), [§7](../rydo-doc/01_PRD.md#7-core-user-journeys)
- Backlog เดิม: ช่องว่างของ RP-020/RP-032 ไม่ใช่พฤติกรรมที่อนุมัติแล้ว
- เกณฑ์รับงาน: เจ้าของสัญญาและผู้ใช้ตกลงวิธีแก้วันที่ก่อนเขียนส่วนนี้; หากเลือกย้ายต้องรักษา snapshot และไม่ทับวันปลายทาง; สเปก/PRD/OpenAPI สอดคล้อง
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ทดสอบตามสัญญาที่ตกลง รวมต้นทุนเปลี่ยน วันซ้ำ concurrency และ rollback; ถ้ายังไม่ตกลงให้คง Blocked ไม่เพิ่ม endpoint เอง
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t14"></a>
### [ ] T14 — Webhook และ durable inbox

- ระยะ: MVP หลังอนุมัติ | สถานะ: Proposed / Not started
- Owner: Backend; ต้องยืนยันผู้รับผิดชอบก่อนเริ่ม
- ขึ้นต่อ: [T12](#t12), [T02](#t02), [T03](#t03)
- SRS: [R12](SRS.md#r12), [R11](SRS.md#r11)
- PRD: [§7](../rydo-doc/01_PRD.md#7-core-user-journeys), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-035–037
- เกณฑ์รับงาน: ตรวจลายเซ็น raw body ก่อนใช้ payload; events ซ้ำมี inbox เดียว; DB commit ก่อน 200; DB ล้มเหลวได้ 503
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: integration test signature ผิด/ถูก batch duplicates และ DB failure; ไม่เปิด route สาธารณะก่อนตรวจผ่าน
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t15"></a>
### [ ] T15 — Relay worker และคำตอบ LINE

- ระยะ: MVP หลังอนุมัติ | สถานะ: Proposed / Not started
- Owner: Backend; ต้องยืนยันผู้รับผิดชอบก่อนเริ่ม
- ขึ้นต่อ: [T14](#t14)
- SRS: [R13](SRS.md#r13), [R11](SRS.md#r11)
- PRD: [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-038–040; RP-042
- เกณฑ์รับงาน: กู้จาก Redis ล่ม/worker หยุดได้; claim/lease/retry มีขอบเขต; สี่คำสั่งตอบ help/ลิงก์ไม่มีจำนวนเงิน; หมดอายุไม่ push แทน
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ทดสอบ duplicate delivery, lease expiry, retry exhaustion และ ambiguous reply; ความล้มเหลวข้อความไม่ undo financial save; Rich Menu เป็น dependency ของเจ้าของ LINE ไม่ใช่ BE แก้เอง
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t16"></a>
### [ ] T16 — ปฏิบัติการและ recovery ของ MVP

- ระยะ: MVP หลังอนุมัติ | สถานะ: Proposed / Not started
- Owner: Backend; ต้องยืนยันผู้รับผิดชอบก่อนเริ่ม
- ขึ้นต่อ: [T12](#t12), [T15](#t15)
- SRS: [R14](SRS.md#r14), [R11](SRS.md#r11)
- PRD: [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- Backlog เดิม: RP-047; ส่วน reliability จากสถาปัตยกรรม
- เกณฑ์รับงาน: มี graceful shutdown, readiness, metrics/backlog/failed jobs และ backup-restore ที่ตรวจแล้ว; retention inbox/response และ privacy ตามนโยบายที่กำหนด
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: ทดสอบ restart/dependency failure/restore ด้วยข้อมูลสมมติ; ตรวจการล้าง reply token และ idempotency expiry ตามสเปก
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

<a id="t17"></a>
### [ ] T17 — ตรวจรับ Backend MVP ร่วมกับผู้เกี่ยวข้อง

- ระยะ: MVP หลังอนุมัติ | สถานะ: Proposed / Not started
- Owner: Backend ร่วมกับ FE/LINE และผู้ตรวจรับ
- ขึ้นต่อ: [T13](#t13), [T15](#t15), [T16](#t16)
- SRS: [R03](SRS.md#r03), [R06](SRS.md#r06), [R07](SRS.md#r07), [R08](SRS.md#r08), [R09](SRS.md#r09), [R10](SRS.md#r10), [R11](SRS.md#r11), [R12](SRS.md#r12), [R13](SRS.md#r13), [R14](SRS.md#r14), [R15](SRS.md#r15)
- PRD: [§7](../rydo-doc/01_PRD.md#7-core-user-journeys), [§9](../rydo-doc/01_PRD.md#9-business-rules), [§9.6](../rydo-doc/01_PRD.md#96-metrics), [§13](../rydo-doc/01_PRD.md#13-acceptance-criteria), [§10](../rydo-doc/01_PRD.md#10-ux-requirements)
- Backlog เดิม: RP-043–048 (ร่วม FE/LINE/pilot owner)
- เกณฑ์รับงาน: หลักฐาน revision เดียวครอบคลุม API acceptance, security และ E2E LINE; Rich Menu RP-041 ส่งต่อเจ้าของ LINE; ช่องว่างต้องแสดง Not checked
- วิธีตรวจ/หลักฐานที่ต้องเก็บ: รัน static/build/tests และ generated-code checks จริง; ส่งผล Passed/Failed/Not checked พร้อม revision; ไม่ถือเอกสารนี้เป็นอนุมัติ deploy
- ผลจริง: ยังไม่มี | Revision/artifact: ยังไม่มี

## ตารางย้อนกลับจากข้อกำหนดสู่งาน

| SRS | Task | PRD |
|---|---|---|
| [R01 — การทดลองและประตูตัดสินใจ](SRS.md#r01) | [T00](#t00), [T10](#t10), [T11](#t11), [T12](#t12) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#11-success-metrics) |
| [R02 — สัญญาและโครงสร้างระบบ](SRS.md#r02) | [T01](#t01), [T02](#t02) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#5-product-goals) |
| [R03 — ตัวตนและ app session](SRS.md#r03) | [T03](#t03), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#7-core-user-journeys) |
| [R04 — โปรไฟล์และแพลตฟอร์ม](SRS.md#r04) | [T04](#t04) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#8-functional-requirements) |
| [R05 — รถและค่าต้นทุน](SRS.md#r05) | [T04](#t04) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#7-core-user-journeys) |
| [R06 — ข้อมูลบันทึกวันทำงาน](SRS.md#r06) | [T01](#t01), [T05](#t05), [T07](#t07), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#9-business-rules) |
| [R07 — การคำนวณ](SRS.md#r07) | [T05](#t05), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#96-metrics) |
| [R08 — Preview ก่อนยืนยัน](SRS.md#r08) | [T06](#t06), [T07](#t07), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#13-acceptance-criteria) |
| [R09 — บันทึกพร้อมกันและส่งซ้ำ](SRS.md#r09) | [T07](#t07), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#13-acceptance-criteria) |
| [R10 — ประวัติและต้นทุนย้อนหลัง](SRS.md#r10) | [T08](#t08), [T13](#t13), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#9-business-rules) |
| [R11 — สิทธิ์และความเป็นส่วนตัว](SRS.md#r11) | [T03](#t03), [T04](#t04), [T06](#t06), [T08](#t08), [T09](#t09), [T14](#t14), [T15](#t15), [T16](#t16), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#13-acceptance-criteria) |
| [R12 — การรับ webhook](SRS.md#r12) | [T14](#t14), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#7-core-user-journeys) |
| [R13 — คิวและคำตอบ LINE](SRS.md#r13) | [T15](#t15), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#13-acceptance-criteria) |
| [R14 — การตรวจและปฏิบัติการ](SRS.md#r14) | [T02](#t02), [T09](#t09), [T10](#t10), [T16](#t16), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#13-acceptance-criteria) |
| [R15 — การเชื่อม LIFF](SRS.md#r15) | [T10](#t10), [T11](#t11), [T17](#t17) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#10-ux-requirements) |
| [R16 — ประเด็นรอตัดสินใจ](SRS.md#r16) | [T00](#t00), [T01](#t01), [T13](#t13) | [หัวข้อต้นทาง](../rydo-doc/01_PRD.md#7-core-user-journeys) |

## บันทึกสถานะและส่งต่อ

- หลักฐานของงานฐานระบบและสูตรคำนวณบันทึกไว้ใน T02 และ T05
- เมื่อรับงาน ให้กรอกผู้รับผิดชอบจริง สถานะ artifact/revision คำสั่งตรวจและผล รวมถึงรายการที่ยังไม่ตรวจใน Task เดิม
- หากข้อกำหนดเปลี่ยน ให้บันทึกแหล่งการตัดสินใจและปรับทั้ง SRS ↔ Task; ห้ามเก็บสถานะ Passed จาก revision เก่าโดยไม่ตรวจผลกระทบ
- ไม่ commit/push/deploy หรือแก้ข้อมูล production จาก checklist นี้ โฟลเดอร์ Backend ยังไม่มี Git repository/branch/commit ณ จัดทำเอกสาร
