# SRS — Backend requirements traceability (ฉบับร่าง)

วันที่รวบรวม: 21 พฤศจิกายน 2026 | อ้าง PRD v1.1

ยังไม่พบ SRS แยกในโครงการก่อนงานนี้ จึงจัดทำฉบับร่างเพื่อเชื่อม Task → ข้อกำหนดระบบ → PRD และสเปกเทคนิคเดิม
รหัส R01–R16 เป็นรหัสอ้างอิงภายในเอกสารนี้ ไม่ใช่รหัสเดิมของ PRD หรือข้อกำหนดที่ได้รับอนุมัติใหม่
เอกสารต้นทางมีอำนาจเหนือบทสรุปนี้เมื่อขัดกัน ประเด็นยังไม่ตกลงให้หยุดเฉพาะงานที่ขึ้นต่อมันและบันทึกการตัดสินใจ
ขอบเขตนี้เป็น Backend และจุดต่อร่วม ไม่ใช่ SRS ทั้งผลิตภัณฑ์หรือการรับรองว่าพร้อมพัฒนา

## ข้อกำหนดและที่มา

<a id="r01"></a>
### R01 — การทดลองและประตูตัดสินใจ

ต้นแบบต้องรองรับทดลอง 7 วันกับ 5 คน โดยวัดการกลับมาใช้เองและผลต่อการตัดสินใจ ก่อนเสนอเริ่ม MVP; ราคาเป็นการทดลองแยกภายหลัง.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#11-success-metrics)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/06_PRODUCT_DECISIONS_AND_RISKS.md#3-pilot-experiment)
- งานที่รองรับ: [T00](Task.md#t00), [T10](Task.md#t10), [T11](Task.md#t11), [T12](Task.md#t12)

<a id="r02"></a>
### R02 — สัญญาและโครงสร้างระบบ

ใช้ Go + Gin, PostgreSQL + sqlc และ OpenAPI ตามสแต็กเดิม; สัญญาต้องระบุ request/response, decimal strings และ error ให้ตรวจได้.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#5-product-goals)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/02_SYSTEM_ARCHITECTURE.md#1-technology-decisions)
- งานที่รองรับ: [T01](Task.md#t01), [T02](Task.md#t02)

<a id="r03"></a>
### R03 — ตัวตนและ app session

ตรวจ LINE ID token ที่ backend; ผูก verified subject กับ user; ตรวจ expiry/revocation ทุก protected request; refresh/logout ตรวจ Origin ตามสัญญา.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#7-core-user-journeys)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#2-authentication)
- งานที่รองรับ: [T03](Task.md#t03), [T17](Task.md#t17)

<a id="r04"></a>
### R04 — โปรไฟล์และแพลตฟอร์ม

อ่าน/แก้ profile และเลือกแพลตฟอร์มประจำได้; ใช้ active codes ไม่ซ้ำ; timezone ของ work date เป็น Asia/Bangkok.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#8-functional-requirements)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#3-current-user)
- งานที่รองรับ: [T04](Task.md#t04)

<a id="r05"></a>
### R05 — รถและค่าต้นทุน

หนึ่งรถ active ต่อบัญชี; ตรวจ efficiency, fuel price และ maintenance reserve; การเปลี่ยนค่ารถไม่เปลี่ยนต้นทุนในรายการเก่า.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#7-core-user-journeys)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#5-vehicles)
- งานที่รองรับ: [T04](Task.md#t04)

<a id="r06"></a>
### R06 — ข้อมูลบันทึกวันทำงาน

หนึ่งรายการ active ต่อคนต่อวัน; รายได้แยกแพลตฟอร์มและไม่ซ้ำ; ระยะทางไม่ติดลบ ชั่วโมง 0–24 และค่าใช้จ่ายไม่รวมน้ำมัน; validate precision/scale ตามฐานข้อมูล.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#9-business-rules)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#6-daily-sessions)
- งานที่รองรับ: [T01](Task.md#t01), [T05](Task.md#t05), [T07](Task.md#t07), [T17](Task.md#t17)

<a id="r07"></a>
### R07 — การคำนวณ

คำนวณน้ำมัน เงินกันซ่อม ต้นทุน กำไร และอัตราด้วย decimal; ปัดต้นทุนแต่ละส่วนเป็นสองตำแหน่ง half away from zero; ตัวหารศูนย์ได้ null.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#96-metrics)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/03_DATABASE_SCHEMA.md#5-money-and-rounding)
- งานที่รองรับ: [T05](Task.md#t05), [T17](Task.md#t17)

<a id="r08"></a>
### R08 — Preview ก่อนยืนยัน

preview ไม่บันทึก; token ผูก user, input, snapshot, target/version และอายุ 15 นาที; การยืนยันต้องตรวจค่าเดิมและปฏิเสธ preview เก่าตามสัญญา.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#6-daily-sessions)
- งานที่รองรับ: [T06](Task.md#t06), [T07](Task.md#t07), [T17](Task.md#t17)

<a id="r09"></a>
### R09 — บันทึกพร้อมกันและส่งซ้ำ

รายการหลักและลูก commit/rollback พร้อมกัน; key เดิมกับคำขอเดิม replay ภายใน 24 ชั่วโมง; key เดิมต่างคำขอ conflict; สร้างพร้อมกันวันเดียวไม่ซ้ำ.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#idempotency-for-create-and-edit)
- งานที่รองรับ: [T07](Task.md#t07), [T17](Task.md#t17)

<a id="r10"></a>
### R10 — ประวัติและต้นทุนย้อนหลัง

ประวัติและ daily report แสดงเฉพาะของตน; edit ใช้ snapshot เดิมและ version; soft delete ไม่รวมในรายงาน; วันที่และรถยังแก้ไม่ได้ตามสัญญาปัจจุบัน.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#9-business-rules)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/03_DATABASE_SCHEMA.md#4-historical-calculation-snapshots)
- งานที่รองรับ: [T08](Task.md#t08), [T13](Task.md#t13), [T17](Task.md#t17)

<a id="r11"></a>
### R11 — สิทธิ์และความเป็นส่วนตัว

ทุก query/mutation จำกัด user; non-owned resource ได้ 404; ไม่ใส่ tokens หรือจำนวนเงินลง analytics/log ที่ไม่จำเป็น; HTTPS, privacy notice และ authenticated financial views ก่อนใช้ข้อมูลจริง.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/02_SYSTEM_ARCHITECTURE.md#8-security-and-privacy)
- งานที่รองรับ: [T03](Task.md#t03), [T04](Task.md#t04), [T06](Task.md#t06), [T08](Task.md#t08), [T09](Task.md#t09), [T14](Task.md#t14), [T15](Task.md#t15), [T16](Task.md#t16), [T17](Task.md#t17)

<a id="r12"></a>
### R12 — การรับ webhook

ตรวจ signature จาก raw body ก่อนเชื่อ payload; deduplicate webhookEventId; commit inbox ก่อนตอบ 200 และตอบ 503 เมื่อรับถาวรไม่ได้.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#7-core-user-journeys)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#8-line-webhook)
- งานที่รองรับ: [T14](Task.md#t14), [T17](Task.md#t17)

<a id="r13"></a>
### R13 — คิวและคำตอบ LINE

relay/worker กู้ pending inbox และ lease หลังล้มเหลว; retry มีขอบเขต; ตอบสี่คำสั่งด้วย help/LIFF link ไม่มีจำนวนเงิน; reply หมดอายุไม่มี push ทดแทน; ข้อความล้มเหลวไม่ย้อนการเงิน.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/02_SYSTEM_ARCHITECTURE.md#6-line-webhook-flow)
- งานที่รองรับ: [T15](Task.md#t15), [T17](Task.md#t17)

<a id="r14"></a>
### R14 — การตรวจและปฏิบัติการ

มี structured logs, health, graceful shutdown, backup/restore และหลักฐาน unit/integration/security/E2E ตามขอบเขตที่จะเปิดใช้; ไม่ถือ build ผ่านว่า API ถูกต้อง.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#13-acceptance-criteria)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/02_SYSTEM_ARCHITECTURE.md#9-reliability)
- งานที่รองรับ: [T02](Task.md#t02), [T09](Task.md#t09), [T10](Task.md#t10), [T16](Task.md#t16), [T17](Task.md#t17)

<a id="r15"></a>
### R15 — การเชื่อม LIFF

ตรวจเส้นทางตั้งค่ารถถึงประวัติบน LINE iOS/Android รวม lost response; frontend เก็บ request/key/token ชั่วคราวตามบัญชีและ retry เดิม; backend รองรับผลตามสัญญา.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#10-ux-requirements)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#11-api-acceptance-tests)
- งานที่รองรับ: [T10](Task.md#t10), [T11](Task.md#t11), [T17](Task.md#t17)

<a id="r16"></a>
### R16 — ประเด็นรอตัดสินใจ

ยังต้องปิดข้อกำหนดวิธีแก้วันที่ที่รักษาต้นทุนเดิม, input/page limits และคำอธิบายชั่วโมงรวม; เป็นช่องว่างหรือข้อเสนอ ไม่ใช่ endpoint/ขีดจำกัดใหม่ที่อนุมัติแล้ว.

- PRD: [หัวข้อที่เกี่ยวข้อง](../rydo-doc/01_PRD.md#7-core-user-journeys)
- สเปกประกอบ: [แหล่งต้นทาง](../rydo-doc/04_API_SPECIFICATION.md#6-daily-sessions)
- งานที่รองรับ: [T00](Task.md#t00), [T01](Task.md#t01), [T13](Task.md#t13)

## ช่องว่างและขอบเขตที่เลื่อนไป

- R16 เป็นรายการเปิด วิธีช่วยแก้วันที่ในต้นแบบเป็นข้อเสนอจากการตรวจธุรกิจ ไม่ใช่สิทธิ์แก้ข้อมูลหรือ API ใหม่
- ต้นแบบไม่ต้องรอ webhook/Redis/worker/Rich Menu ตามข้อเสนอใน [แผน](IMPLEMENTATION_PLAN.md); งานเหล่านี้ยังเป็น P0 ของ MVP ตาม PRD
- หลักฐานและค่าตั้งต้นรถที่นำมาแนะนำต้องมีแหล่งและตรวจความเข้าใจกับไรเดอร์ ไม่ตั้งตัวเลขแทนธุรกิจเอง
- งาน frontend, mobile UX, Rich Menu และ LINE Console ต้องมีเจ้าของร่วม; การมี Backend task ไม่แปลว่าฝั่งอื่นเสร็จ
- นอกขอบเขต MVP ตาม [PRD §6](../rydo-doc/01_PRD.md#6-non-goals-for-mvp): platform integration, GPS, OCR, payments, actual-fuel accounting และ server-side drafts

## การปรับปรุงการเชื่อมโยง

เมื่อ PRD/API/Schema เปลี่ยน ให้ตรวจ R ที่ได้รับผล แล้วตามลิงก์ไป Task และหลักฐานทดสอบ
เมื่อทำ Task ให้ส่ง revision/ผลตรวจกลับมาที่ R ที่รองรับ โดยไม่ทำเครื่องหมายผ่านจากการเขียนเอกสารอย่างเดียว
