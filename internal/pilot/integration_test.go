package pilot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testDB *pgxpool.Pool
var captured struct {
	sync.Mutex
	Rows []any
}

// Tests always create their own isolated PostgreSQL cluster. No DATABASE_URL input is used.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	dir, err := os.MkdirTemp("", "rydo-pilot-pg-")
	if err != nil {
		panic(err)
	}
	bin, err := exec.Command("pg_config", "--bindir").Output()
	if err != nil {
		panic(err)
	}
	pg := strings.TrimSpace(string(bin))
	run := func(name string, args ...string) {
		out, e := exec.Command(filepath.Join(pg, name), args...).CombinedOutput()
		if e != nil {
			panic(fmt.Sprintf("%s: %s: %v", name, out, e))
		}
	}
	run("initdb", "-D", dir+"/db", "-A", "trust", "--no-locale", "--encoding=UTF8")
	run("pg_ctl", "-D", dir+"/db", "-l", dir+"/postgres.log", "-o", "-c listen_addresses= -k "+dir, "-w", "start")
	cleanup := func() {
		exec.Command(filepath.Join(pg, "pg_ctl"), "-D", dir+"/db", "-m", "fast", "-w", "stop").Run()
		os.RemoveAll(dir)
	}
	testDB, err = pgxpool.New(context.Background(), "postgres:///postgres?host="+dir+"&sslmode=disable")
	if err != nil {
		cleanup()
		panic(err)
	}
	files, _ := filepath.Glob("../../db/migrations/*.up.sql")
	for _, file := range files {
		body, e := os.ReadFile(file)
		if e != nil {
			panic(e)
		}
		if _, e = testDB.Exec(context.Background(), string(body)); e != nil {
			testDB.Close()
			cleanup()
			panic(e)
		}
	}
	code := m.Run()
	if path := os.Getenv("RYDO_CAPTURE_HTTP"); path != "" {
		raw, e := json.MarshalIndent(captured.Rows, "", "  ")
		if e != nil {
			panic(e)
		}
		if e = os.WriteFile(path, raw, 0600); e != nil {
			panic(e)
		}
	}
	testDB.Close()
	cleanup()
	os.Exit(code)
}

type fixture struct {
	s     *Server
	url   string
	clock *atomic.Int64
	t     *testing.T
}
type reply struct {
	status int
	body   map[string]any
	header http.Header
	raw    []byte
}

func setup(t *testing.T) *fixture {
	t.Helper()
	if _, err := testDB.Exec(context.Background(), "TRUNCATE users CASCADE"); err != nil {
		t.Fatal(err)
	}
	api, err := New(testDB, Config{Origin: "https://pilot.example.test", JWTKey: "test-access-key-01234567890123456789", PreviewKey: "test-preview-key-01234567890123456789", LineChannelID: "test-channel"})
	if err != nil {
		t.Fatal(err)
	}
	now := new(atomic.Int64)
	now.Store(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC).Unix())
	api.now = func() time.Time { return time.Unix(now.Load(), 0) }
	// Only this _test.go file replaces the external LINE verification boundary.
	api.verify = func(_ context.Context, token string) (string, error) {
		switch token {
		case "line-A":
			return "synthetic-line-A", nil
		case "line-B":
			return "synthetic-line-B", nil
		default:
			return "", errors.New("invalid test ID token")
		}
	}
	router := gin.New()
	api.Register(router)
	httpServer := httptest.NewServer(router)
	t.Cleanup(httpServer.Close)
	return &fixture{api, httpServer.URL, now, t}
}
func (f *fixture) request(method, path, token string, body any, headers map[string]string) reply {
	f.t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			f.t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, f.url+path, &b)
	if err != nil {
		f.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://pilot.example.test")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	defer response.Body.Close()
	var raw bytes.Buffer
	raw.ReadFrom(response.Body)
	parsed := map[string]any{}
	if raw.Len() > 0 {
		if err := json.Unmarshal(raw.Bytes(), &parsed); err != nil {
			f.t.Fatalf("non JSON %d %s: %s", response.StatusCode, path, raw.String())
		}
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		f.t.Fatalf("missing no-store: %s", path)
	}
	// Save only synthetic, redacted wire shapes for independent contract validation.
	rawRecord, _ := json.Marshal(map[string]any{"method": method, "path": path, "status": response.StatusCode, "request": body, "response": parsed, "cache_control": response.Header.Get("Cache-Control"), "set_cookie_present": response.Header.Get("Set-Cookie") != ""})
	var record map[string]any
	json.Unmarshal(rawRecord, &record)
	var redact func(any)
	redact = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				if k == "access_token" || k == "preview_token" || k == "id_token" {
					x[k] = "SYNTHETIC_REDACTED"
				} else {
					redact(v)
				}
			}
		case []any:
			for _, v := range x {
				redact(v)
			}
		}
	}
	redact(record)
	captured.Lock()
	captured.Rows = append(captured.Rows, record)
	captured.Unlock()
	return reply{response.StatusCode, parsed, response.Header, raw.Bytes()}
}
func want(t *testing.T, r reply, status int) reply {
	t.Helper()
	if r.status != status {
		t.Fatalf("want %d got %d: %s", status, r.status, r.raw)
	}
	return r
}
func data(r reply) map[string]any { return r.body["data"].(map[string]any) }
func (f *fixture) login(who string) (string, string) {
	f.t.Helper()
	r := want(f.t, f.request("POST", "/api/v1/auth/line", "", map[string]any{"id_token": who}, nil), 200)
	cookies := (&http.Response{Header: r.header}).Cookies()
	if len(cookies) != 1 {
		f.t.Fatal("expected refresh cookie")
	}
	c := cookies[0]
	if !c.Secure || !c.HttpOnly || c.Path != "/" || c.Domain != "" || c.SameSite != http.SameSiteLaxMode {
		f.t.Fatalf("unsafe cookie: %+v", c)
	}
	return data(r)["access_token"].(string), c.Name + "=" + c.Value
}
func vehicleBody() map[string]any {
	return map[string]any{"name": "Test bike", "km_per_liter": "40.000", "fuel_price_per_liter": "35.00", "maintenance_reserve_per_km": "0.600", "is_default": true}
}
func (f *fixture) vehicle(token string) string {
	r := want(f.t, f.request("POST", "/api/v1/vehicles", token, vehicleBody(), nil), 201)
	return data(r)["id"].(string)
}

// Breaks if origin checking, LINE rejection, session revocation, or expiry is omitted.
func TestAuthenticationSessionAndOrigin(t *testing.T) {
	f := setup(t)
	want(t, f.request("POST", "/api/v1/auth/line", "", map[string]any{"id_token": "bad"}, nil), 401)
	want(t, f.request("POST", "/api/v1/auth/line", "", map[string]any{"id_token": "line-A"}, map[string]string{"Origin": "https://pilot.example.test.evil"}), 403)
	token, cookie := f.login("line-A")
	me := want(t, f.request("GET", "/api/v1/me", token, nil, nil), 200)
	if data(me)["onboarding_completed"] != false {
		t.Fatal("new user unexpectedly onboarded")
	}
	want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie, "Origin": ""}), 403)
	f.clock.Add(901)
	want(t, f.request("GET", "/api/v1/me", token, nil, nil), 401)
	refreshed := want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 200)
	token = data(refreshed)["access_token"].(string)
	want(t, f.request("GET", "/api/v1/me", token, nil, nil), 200)
	want(t, f.request("POST", "/api/v1/auth/logout", "", nil, map[string]string{"Cookie": cookie}), 204)
	want(t, f.request("GET", "/api/v1/me", token, nil, nil), 401)
	want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 401)
	want(t, f.request("POST", "/api/v1/auth/logout", "", nil, nil), 204)
	token, cookie = f.login("line-A")
	f.clock.Add(30 * 86400)
	want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 401)
	want(t, f.request("GET", "/api/v1/me", token, nil, nil), 401)
}

// Breaks if owner scopes, database uniqueness, or onboarding derivation is removed.
func TestProfilesVehiclesAndOwnership(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	b, _ := f.login("line-B")
	v := f.vehicle(a)
	want(t, f.request("POST", "/api/v1/vehicles", a, vehicleBody(), nil), 409)
	want(t, f.request("PATCH", "/api/v1/vehicles/"+v, b, map[string]any{"name": "stolen"}, nil), 404)
	want(t, f.request("PATCH", "/api/v1/vehicles/"+v, a, map[string]any{"fuel_price_per_liter": "36.00"}, nil), 200)
	profile := map[string]any{"display_name": "Rider A", "timezone": "Asia/Bangkok", "preferred_platform_codes": []string{"grab", "lineman"}}
	r := want(t, f.request("PUT", "/api/v1/me/profile", a, profile, nil), 200)
	if data(r)["onboarding_completed"] != true {
		t.Fatal("setup not completed")
	}
	r = want(t, f.request("GET", "/api/v1/vehicles", b, nil, nil), 200)
	if len(r.body["data"].([]any)) != 0 {
		t.Fatal("vehicle exposed to another user")
	}
	want(t, f.request("GET", "/api/v1/platforms", a, nil, nil), 200)
	want(t, f.request("GET", "/api/v1/me/profile", a, nil, nil), 200)
	want(t, f.request("GET", "/api/v1/vehicles", a, nil, nil), 200)
	profile["preferred_platform_codes"] = []string{"grab", "grab"}
	want(t, f.request("PUT", "/api/v1/me/profile", a, profile, nil), 422)
	for _, patch := range []any{map[string]any{}, map[string]any{"name": nil}, map[string]any{"km_per_liter": "0"}, map[string]any{"fuel_price_per_liter": 3}, map[string]any{"is_default": false}} {
		want(t, f.request("PATCH", "/api/v1/vehicles/"+v, a, patch, nil), 422)
	}
}

func dayBody(vehicle string) map[string]any {
	return map[string]any{"work_date": "2026-11-21", "vehicle_id": vehicle, "distance_km": "100.00", "working_hours": "8.00", "incomes": []any{map[string]any{"platform_code": "grab", "gross_income": "850.00"}}, "expenses": []any{map[string]any{"category": "parking", "amount": "30.00", "note": ""}}, "note": ""}
}
func (f *fixture) preview(token, path string, body any) string {
	r := want(f.t, f.request("POST", path, token, body, nil), 200)
	return data(r)["preview_token"].(string)
}
func confirmHeaders(preview, key string) map[string]string {
	return map[string]string{"X-Preview-Token": preview, "Idempotency-Key": key}
}
func assertSummary(t *testing.T, r reply, net, perHour string) {
	t.Helper()
	summary := data(r)["summary"].(map[string]any)
	if summary["estimated_net_profit"] != net || summary["estimated_net_per_hour"] != perHour {
		t.Fatalf("wrong summary %v", summary)
	}
}

// Breaks if preview writes, save loses nested data, replay revalidates expired previews,
// or an old record is recalculated with current vehicle settings.
func TestPreviewConfirmationReplayAndHistoricalSnapshot(t *testing.T) {
	f := setup(t)
	a, cookie := f.login("line-A")
	b, _ := f.login("line-B")
	v := f.vehicle(a)
	body := dayBody(v)
	preview := f.preview(a, "/api/v1/sessions/preview", body)
	var count int
	if err := testDB.QueryRow(context.Background(), "SELECT count(*) FROM daily_sessions").Scan(&count); err != nil || count != 0 {
		t.Fatal("preview wrote a record", err, count)
	}
	headers := confirmHeaders(preview, "44444444-4444-4444-8444-444444444444")
	r := want(t, f.request("POST", "/api/v1/sessions", a, body, headers), 201)
	assertSummary(t, r, "672.50", "84.06")
	sid := data(r)["id"].(string)
	original := r.body
	want(t, f.request("GET", "/api/v1/sessions/"+sid, b, nil, nil), 404)
	want(t, f.request("POST", "/api/v1/sessions/preview", b, body, nil), 404)
	want(t, f.request("PATCH", "/api/v1/sessions/"+sid, b, map[string]any{"note": "steal"}, headers), 404)
	r = want(t, f.request("GET", "/api/v1/sessions/"+sid, a, nil, nil), 200)
	if data(r)["vehicle_id"] != v || len(data(r)["incomes"].([]any)) != 1 {
		t.Fatal("missing saved fields")
	}
	want(t, f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(preview, "55555555-5555-4555-8555-555555555555")), 409)
	f.clock.Add(901)
	a = data(want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 200))["access_token"].(string)
	r = want(t, f.request("POST", "/api/v1/sessions", a, body, headers), 201)
	old, _ := json.Marshal(original)
	now, _ := json.Marshal(r.body)
	if !bytes.Equal(old, now) {
		t.Fatal("replay changed response")
	}
	body["note"] = "changed"
	want(t, f.request("POST", "/api/v1/sessions", a, body, headers), 409)
	want(t, f.request("PATCH", "/api/v1/vehicles/"+v, a, map[string]any{"fuel_price_per_liter": "70.00"}, nil), 200)
	patch := map[string]any{"working_hours": "4.00"}
	ep := f.preview(a, "/api/v1/sessions/"+sid+"/preview", patch)
	r = want(t, f.request("PATCH", "/api/v1/sessions/"+sid, a, patch, confirmHeaders(ep, "66666666-6666-4666-8666-666666666666")), 200)
	assertSummary(t, r, "672.50", "168.13")
	if data(r)["version"] != float64(2) {
		t.Fatal("version not incremented")
	}
	want(t, f.request("PATCH", "/api/v1/sessions/"+sid, a, map[string]any{"work_date": "2026-11-20"}, headers), 422)
	r = want(t, f.request("GET", "/api/v1/sessions?from=2026-11-21&to=2026-11-21&page=1&page_size=1", a, nil, nil), 200)
	if len(r.body["data"].([]any)) != 1 || r.body["meta"].(map[string]any)["has_more"] != false {
		t.Fatal("history missing")
	}
	r = want(t, f.request("GET", "/api/v1/reports/daily?date=2026-11-20", a, nil, nil), 200)
	if r.body["data"] != nil {
		t.Fatal("empty report should be null")
	}
}

// Breaks if date uniqueness, idempotency serialization, version locking or rollback is missing.
func TestConcurrentConfirmAndTransactionalRollback(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	v := f.vehicle(a)
	body := dayBody(v)
	p := f.preview(a, "/api/v1/sessions/preview", body)
	headers := confirmHeaders(p, "77777777-7777-4777-8777-777777777777")
	responses := make(chan reply, 6)
	for i := 0; i < 6; i++ {
		go func() { responses <- f.request("POST", "/api/v1/sessions", a, body, headers) }()
	}
	var sid string
	for i := 0; i < 6; i++ {
		r := want(t, <-responses, 201)
		if sid != "" && sid != data(r)["id"] {
			t.Fatal("same key created multiple records")
		}
		sid = data(r)["id"].(string)
	}
	var sessions, incomes int
	testDB.QueryRow(context.Background(), "SELECT count(*) FROM daily_sessions").Scan(&sessions)
	testDB.QueryRow(context.Background(), "SELECT count(*) FROM income_entries").Scan(&incomes)
	if sessions != 1 || incomes != 1 {
		t.Fatal("duplicate write", sessions, incomes)
	}
	patch := map[string]any{"note": "edit"}
	p = f.preview(a, "/api/v1/sessions/"+sid+"/preview", patch)
	want(t, f.request("PATCH", "/api/v1/sessions/"+sid, a, patch, confirmHeaders(p, "88888888-8888-4888-8888-888888888888")), 200)
	want(t, f.request("PATCH", "/api/v1/sessions/"+sid, a, patch, confirmHeaders(p, "99999999-9999-4999-8999-999999999999")), 409)
	// A DB trigger fails after the session write, proving the entire transaction rolls back.
	_, err := testDB.Exec(context.Background(), `CREATE FUNCTION fail_test_expense() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic DB failure'; END $$; CREATE TRIGGER fail_test_expense BEFORE INSERT ON expense_entries FOR EACH ROW EXECUTE FUNCTION fail_test_expense()`)
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Exec(context.Background(), `DROP TRIGGER fail_test_expense ON expense_entries; DROP FUNCTION fail_test_expense()`)
	body["work_date"] = "2026-11-20"
	p = f.preview(a, "/api/v1/sessions/preview", body)
	want(t, f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(p, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")), 503)
	testDB.QueryRow(context.Background(), "SELECT count(*) FROM daily_sessions").Scan(&sessions)
	if sessions != 1 {
		t.Fatal("partial save survived failure")
	}
	var requests int
	testDB.QueryRow(context.Background(), "SELECT count(*) FROM session_requests WHERE request_key='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'").Scan(&requests)
	if requests != 0 {
		t.Fatal("failed save persisted success replay")
	}
}

func TestStalePreviewAndInputValidation(t *testing.T) {
	f := setup(t)
	a, cookie := f.login("line-A")
	v := f.vehicle(a)
	body := dayBody(v)
	for _, field := range []string{"distance_km", "working_hours"} {
		old := body[field]
		for _, value := range []any{nil, "-1", "NaN", "1e2", 3.14, "0.001"} {
			body[field] = value
			want(t, f.request("POST", "/api/v1/sessions/preview", a, body, nil), 422)
		}
		body[field] = old
	}
	body["working_hours"] = "24.01"
	want(t, f.request("POST", "/api/v1/sessions/preview", a, body, nil), 422)
	body["working_hours"] = "8.00"
	body["work_date"] = "2026-11-22"
	want(t, f.request("POST", "/api/v1/sessions/preview", a, body, nil), 422)
	body["work_date"] = "2026-11-21"
	p := f.preview(a, "/api/v1/sessions/preview", body)
	want(t, f.request("POST", "/api/v1/sessions", a, body, nil), 422)
	want(t, f.request("PATCH", "/api/v1/vehicles/"+v, a, map[string]any{"fuel_price_per_liter": "36.00"}, nil), 200)
	want(t, f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(p, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")), 409)
	p = f.preview(a, "/api/v1/sessions/preview", body)
	f.clock.Add(901)
	a = data(want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 200))["access_token"].(string)
	want(t, f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(p, "cccccccc-cccc-4ccc-8ccc-cccccccccccc")), 409)
}

// Regression: a wrongly typed date must be a client error, not a service failure.
func TestRejectsNonStringWorkDate(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	body := dayBody(f.vehicle(a))
	body["work_date"] = 20261121
	want(t, f.request("POST", "/api/v1/sessions/preview", a, body, nil), 422)
}

func TestDifferentKeysSameDateAndVersionConcurrency(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	v := f.vehicle(a)
	body := dayBody(v)
	p := f.preview(a, "/api/v1/sessions/preview", body)
	results := make(chan reply, 2)
	for _, key := range []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"} {
		go func(k string) { results <- f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(p, k)) }(key)
	}
	success, conflicts := 0, 0
	var sid string
	for range 2 {
		r := <-results
		if r.status == 201 {
			success++
			sid = data(r)["id"].(string)
		} else if r.status == 409 {
			conflicts++
			details := r.body["error"].(map[string]any)["details"].([]any)
			if len(details) == 0 {
				t.Fatal("date conflict missing own session_id")
			}
		} else {
			t.Fatalf("unexpected %d %s", r.status, r.raw)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("different keys duplicated a date", success, conflicts)
	}
	patch := map[string]any{"note": "concurrent"}
	p = f.preview(a, "/api/v1/sessions/"+sid+"/preview", patch)
	for _, key := range []string{"33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"} {
		go func(k string) { results <- f.request("PATCH", "/api/v1/sessions/"+sid, a, patch, confirmHeaders(p, k)) }(key)
	}
	success, conflicts = 0, 0
	for range 2 {
		r := <-results
		if r.status == 200 {
			success++
		} else if r.status == 409 {
			conflicts++
		} else {
			t.Fatal("unexpected edit", r.status)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("stale version applied twice")
	}
	r := want(t, f.request("GET", "/api/v1/sessions/"+sid, a, nil, nil), 200)
	if data(r)["version"] != float64(2) {
		t.Fatal("wrong concurrent version")
	}
}

func TestZeroRatesPaginationInactivePlatformAndDelete(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	b, _ := f.login("line-B")
	v := f.vehicle(a)
	body := dayBody(v)
	body["distance_km"] = "0.00"
	body["working_hours"] = "0.00"
	p := f.preview(a, "/api/v1/sessions/preview", body)
	r := want(t, f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(p, "11111111-1111-4111-8111-111111111111")), 201)
	sid := data(r)["id"].(string)
	want(t, f.request("GET", "/api/v1/reports/daily?date=2026-11-21", a, nil, nil), 200)
	summary := data(r)["summary"].(map[string]any)
	if summary["estimated_net_per_hour"] != nil || summary["estimated_net_per_km"] != nil || summary["estimated_net_profit"] != "820.00" {
		t.Fatal("wrong zero rates", summary)
	}
	body["work_date"] = "2026-11-20"
	p = f.preview(a, "/api/v1/sessions/preview", body)
	want(t, f.request("POST", "/api/v1/sessions", a, body, confirmHeaders(p, "22222222-2222-4222-8222-222222222222")), 201)
	r = want(t, f.request("GET", "/api/v1/sessions?page_size=1", a, nil, nil), 200)
	if r.body["meta"].(map[string]any)["has_more"] != true {
		t.Fatal("pagination stopped early")
	}
	r = want(t, f.request("GET", "/api/v1/sessions?page_size=1&page=2", a, nil, nil), 200)
	if r.body["meta"].(map[string]any)["has_more"] != false {
		t.Fatal("pagination never ends")
	}
	r = want(t, f.request("GET", "/api/v1/sessions", b, nil, nil), 200)
	if len(r.body["data"].([]any)) != 0 {
		t.Fatal("history exposed to another user")
	}
	want(t, f.request("GET", "/api/v1/sessions?page_size=101", a, nil, nil), 422)
	testDB.Exec(context.Background(), "UPDATE platforms SET is_active=false WHERE code='grab'")
	defer testDB.Exec(context.Background(), "UPDATE platforms SET is_active=true WHERE code='grab'")
	patch := map[string]any{"note": "retain inactive"}
	p = f.preview(a, "/api/v1/sessions/"+sid+"/preview", patch)
	want(t, f.request("PATCH", "/api/v1/sessions/"+sid, a, patch, confirmHeaders(p, "33333333-3333-4333-8333-333333333333")), 200)
	body["work_date"] = "2026-11-19"
	want(t, f.request("POST", "/api/v1/sessions/preview", a, body, nil), 422)
	want(t, f.request("DELETE", "/api/v1/sessions/"+sid, b, nil, nil), 404)
	want(t, f.request("DELETE", "/api/v1/sessions/"+sid, a, nil, nil), 204)
	want(t, f.request("GET", "/api/v1/sessions/"+sid, a, nil, nil), 404)
	r = want(t, f.request("GET", "/api/v1/reports/daily?date=2026-11-21", a, nil, nil), 200)
	if r.body["data"] != nil {
		t.Fatal("deleted day still in report")
	}
}

func TestRefreshDoesNotExtendSessionAndReplacedCookieIsRevoked(t *testing.T) {
	f := setup(t)
	a, old := f.login("line-A")
	r := want(t, f.request("POST", "/api/v1/auth/line", "", map[string]any{"id_token": "line-A"}, map[string]string{"Cookie": old}), 200)
	cookie := (&http.Response{Header: r.header}).Cookies()[0].String()
	want(t, f.request("GET", "/api/v1/me", a, nil, nil), 401)
	want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": old}), 401)
	f.clock.Add(30*86400 - 30)
	r = want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 200)
	if data(r)["expires_in"] != float64(30) {
		t.Fatal("access expiry exceeds original session lifetime")
	}
	f.clock.Add(31)
	want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 401)
}

func TestRefreshSetsOnlyRemainingCookieLifetime(t *testing.T) {
	f := setup(t)
	_, cookie := f.login("line-A")
	f.clock.Add(60)
	r := want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 200)
	cookies := (&http.Response{Header: r.header}).Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != 30*86400-60 {
		t.Fatal("refresh must preserve remaining original cookie lifetime")
	}
}

func TestPreviewCannotCrossUsersAndKeysAreUserScoped(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	b, _ := f.login("line-B")
	da := dayBody(f.vehicle(a))
	db := dayBody(f.vehicle(b))
	pa := f.preview(a, "/api/v1/sessions/preview", da)
	key := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	want(t, f.request("POST", "/api/v1/sessions", b, db, confirmHeaders(pa, key)), 409)
	want(t, f.request("POST", "/api/v1/sessions", a, da, confirmHeaders(pa, key)), 201)
	pb := f.preview(b, "/api/v1/sessions/preview", db)
	want(t, f.request("POST", "/api/v1/sessions", b, db, confirmHeaders(pb, key)), 201)
	parts := strings.Split(a, ".")
	parts[2] = "tampered-signature"
	want(t, f.request("GET", "/api/v1/me", strings.Join(parts, "."), nil, nil), 401)
	f.clock.Add(24*3600 + 1)
	_, cookie := f.login("line-A")
	a = data(want(t, f.request("POST", "/api/v1/auth/refresh", "", nil, map[string]string{"Cookie": cookie}), 200))["access_token"].(string)
	want(t, f.request("POST", "/api/v1/sessions", a, da, confirmHeaders(pa, key)), 409)
}

func TestRejectsUnknownCompoundPropertyNames(t *testing.T) {
	f := setup(t)
	a, _ := f.login("line-A")
	v := f.vehicle(a)
	want(t, f.request("PATCH", "/api/v1/vehicles/"+v, a, map[string]any{"name km_per_liter": "ignored"}, nil), 422)
}
