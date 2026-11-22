# T01 verification and handoff

วันที่ตรวจ: 2026-11-21T23:12:15+07:00
สถานะส่งมอบ: **Partial — independent proposed pilot contract ready for review; D01 remains open**

## Revision

- OpenAPI SHA-256: `d2478bc1a9328e2379e5a5655208ad4cc1015c2ce4cd2c68ff07f6c2da21d9af`
- ยังไม่มี Git repository ใน Backend จึงไม่มี branch/commit; ใช้ artifact hash กับ source-reviewed.json ผูกหลักฐาน
- source-baseline.json เก็บสภาพแรก; source-reviewed.json เก็บหลังอ่าน migration000003 และ readiness schema3 ที่เปลี่ยนโดยงานอื่น
- validator ยืนยัน input files 13 ไฟล์ตาม source-reviewed ไม่เปลี่ยนระหว่างรอบตรวจสุดท้าย (ยกเว้น Task.md ที่แก้ผล T01)

## ผลที่รันจริง

| รายการ | ผล | หลักฐาน |
|---|---|---|
| OpenAPI 3.1 validation + local references | Passed | openapi-spec-validator0.9.0 / jsonschema4.26.0 |
| Operation coverage | Passed | 18 operations / 13 paths; every operation marked proposed/not implemented |
| Schema/example checks | Passed | 218 checks รวม47 positive/negative cases, inline requests/responses/errors และ confirmation retry headers |
| ตัวอย่างจาก API ต้นทาง | Passed | parse JSON10ชุด; validate8ชุดของ pilot (แทน placeholder uuidด้วย UUIDสมมติ); ไม่ใช้ generic envelopeและpost-MVP preferencesเป็น endpointตัวอย่าง |
| สูตรอิสระ | Passed | 87.50 fuel,60.00 reserve,672.50 net,84.06/hour,6.73/km; ties±6.725; edit7.75h/expense[]ได้702.50 net,90.65/hour |
| Go static | Passed exit0 | `/usr/local/go/bin/go vet ./...`; go-vet.log |
| Go tests | Passed exit0 | `/usr/local/go/bin/go test ./...`; go-test.log (Go cache reused where inputs unchanged) |
| Go build | Passed exit0 | `/usr/local/go/bin/go build -buildvcs=false ./...`; go-build.log |
| Runtime Swagger source separation | Passed | hashเดิม, pathsมีแค่ health/live และ health/ready |
| Live localhost8080 | Not checked successfully | curl `/swagger/openapi.json` ได้connection refused (exit7); ไม่เปิด serverขึ้นแทนงานเจ้าของ runtime |
| Runtime pilot endpoints / LINE / DB integration | Not checked | ไม่ implement T02–T08, ไม่แตะบัญชีLINE ไม่ deploy |

Go ที่ใช้จริง: go1.25.0 linux/amd64 ผ่าน `/usr/local/go/bin/go` (คำสั่ง `go` แรกไม่อยู่ใน PATH; ใช้ binaryที่มีอยู่และ toolchainตาม go.mod)
การผ่านGo checksเป็น baselineของโค้ดที่มีอยู่ ไม่ใช่ผลตรวจ18 operationsที่ยังไม่ implement

## Scope and acceptance

- ส่งมอบ auth/session, profile, vehicle, platforms, create/edit preview + confirmation, history/detail/edit/deleteและdaily reportตาม API§2–7
- exact contract response/validation/error/provisional decisions P01–P08 และ mapping frontend M-03..M-19 อยู่ใน README
- ไม่เปลี่ยน PRD/API/Frontend/Go/migration/runtime Swagger; แก้ Task.mdเฉพาะส่วนT01 และเพิ่ม contracts/t01
- ไม่เพิ่มงานแก้วันที่ใน API สเปกยัง immutable; D01 ต้องให้Product/เจ้าของAPI/ผู้ใช้กำหนด workflowและสิทธิ์ก่อนT10
- P01–P08 เป็นค่าที่เสนอย้อนเปลี่ยนได้ ต้องให้เจ้าของAPI/Frontendรับbaselineก่อนfreeze adapter
- D02 (HTTPS/LIFF/origin/provider) ขวางLINE E2Eจริง แต่ไม่ขวางการ implement/test ด้วยข้อมูลสมมติ

## Next owner

Rika รับ proposed baselineและประเด็นD01/D02; เจ้าของ Backendตรวจสิ่งที่T02มีแล้วและทำเฉพาะส่วนขาดในT03–T08
Frontendเตรียม typed adapterจากschema: strip immutable fieldsตอนPATCH, hydrate detailหลังconfirm, paginate has_more,
ผูกaccountIdกับverified /me และรักษาpayload/key/tokenเดิมเมื่อlost response ไม่ถือfixture30/30เป็นintegrationผ่าน
ขั้นตอนละเอียดและเงื่อนไขความพร้อมก่อนข้อมูลจริงอยู่ในREADME

## Repeat

```sh
uv run --with openapi-spec-validator==0.9.0 --with jsonschema==4.26.0 python contracts/t01/validate_contract.py
```

เครื่องมืออยู่ใน uv cache แยกจาก dependenciesผลิตภัณฑ์ ไม่แก้go.mod/go.sum
ไม่มี commit/push/deploy หรือการแก้บัญชีLINEจริง
