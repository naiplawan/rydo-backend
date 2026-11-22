package pilot

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type previewClaims struct {
	User       string `json:"user"`
	Target     string `json:"target"`
	Version    int64  `json:"version"`
	InputHash  string `json:"input"`
	MergedHash string `json:"merged"`
	Efficiency string `json:"efficiency"`
	Price      string `json:"price"`
	Reserve    string `json:"reserve"`
	Expires    int64  `json:"exp"`
}

func stale(reason string) error { return conflict("PREVIEW_STALE", gin.H{"reason": reason}) }
func (s *Server) readInput(c *gin.Context, edit bool) ([]byte, error) {
	allowed, required := createFields, createRequired
	if edit {
		allowed, required = editFields, ""
	}
	m, e := body(c, allowed, required)
	if e != nil {
		return nil, e
	}
	if edit && len(m) == 0 {
		return nil, bad("body")
	}
	return canonicalMap(m, !edit)
}
func (s *Server) loadTarget(c *gin.Context, q queryer, input []byte, sid string, lock bool) (record, []Income, error) {
	var r record
	var previous []Income
	var e error
	if sid != "" {
		r, e = readRecord(c.Request.Context(), q, uid(c), sid, lock)
		if e != nil {
			return r, nil, e
		}
		previous = r.Incomes
		r.Workday, e = mergedInput(r.Workday, input)
	} else {
		e = json.Unmarshal(input, &r.Workday)
		if e == nil {
			var v Vehicle
			v, e = readVehicle(c.Request.Context(), q, uid(c), r.Vehicle, lock)
			r.efficiency, r.price, r.reserve = v.Efficiency, v.Price, v.Reserve
		}
	}
	if e != nil {
		return r, nil, e
	}
	if e = s.validateDay(r.Workday); e != nil {
		return r, nil, e
	}
	return r, previous, nil
}
func (s *Server) preview(c *gin.Context) error {
	sid := ""
	var e error
	if c.Param("session_id") != "" {
		sid, e = id(c, "session_id")
		if e != nil {
			return e
		}
	}
	input, e := s.readInput(c, sid != "")
	if e != nil {
		return e
	}
	var claims previewClaims
	var value Summary
	// Read a consistent snapshot of the record, child entries, and vehicle settings.
	tx, e := s.db.BeginTx(c.Request.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Request.Context())
	r, previous, e := s.loadTarget(c, tx, input, sid, false)
	if e != nil {
		return e
	}
	// Read-only transactions cannot SELECT FOR SHARE; validation uses this snapshot.
	for _, i := range r.Incomes {
		var active bool
		e = tx.QueryRow(c.Request.Context(), `SELECT is_active FROM platforms WHERE code=$1`, i.Platform).Scan(&active)
		if errors.Is(e, pgx.ErrNoRows) {
			return bad("incomes")
		}
		if e != nil {
			return e
		}
		saved := false
		for _, old := range previous {
			if old.Platform == i.Platform {
				saved = true
			}
		}
		if !active && !saved {
			return bad("incomes")
		}
	}
	value, e = summary(r.Workday, r.efficiency, r.price, r.reserve)
	if e != nil {
		return e
	}
	claims = previewClaims{uid(c), sid, r.Version, digest(input), mergedHash(r.Workday), normalizedDecimal(r.efficiency), normalizedDecimal(r.price), normalizedDecimal(r.reserve), s.now().Add(15 * time.Minute).Unix()}
	if e = tx.Commit(c.Request.Context()); e != nil {
		return e
	}
	ok(c, 200, gin.H{"summary": value, "preview_token": signed(claims, s.config.PreviewKey)})
	return nil
}
func (s *Server) confirm(c *gin.Context) error {
	sid := ""
	var e error
	if c.Param("session_id") != "" {
		sid, e = id(c, "session_id")
		if e != nil {
			return e
		}
	}
	input, e := s.readInput(c, sid != "")
	if e != nil {
		return e
	}
	key := c.GetHeader("Idempotency-Key")
	token := c.GetHeader("X-Preview-Token")
	if !uuidRE.MatchString(key) {
		return bad("Idempotency-Key")
	}
	key = strings.ToLower(key)
	if token == "" || len(token) > 16384 {
		return bad("X-Preview-Token")
	}
	fingerprint := digest([]byte(c.Request.Method + "\n" + strings.ToLower(c.Request.URL.Path) + "\n" + string(input) + "\n" + token))
	status := 201
	if sid != "" {
		status = 200
	}
	var response []byte
	e = s.transaction(c, func(tx pgx.Tx) error {
		ctx := c.Request.Context()
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, uid(c)+":"+key); err != nil {
			return err
		}
		var storedFingerprint, storedResponse []byte
		var storedStatus int
		err := tx.QueryRow(ctx, `SELECT request_fingerprint,response_status,response_body::text FROM session_requests WHERE user_id=$1 AND request_key=$2 AND expires_at>$3`, uid(c), key, s.now()).Scan(&storedFingerprint, &storedStatus, &storedResponse)
		if err == nil {
			if !bytes.Equal(storedFingerprint, []byte(fingerprint)) {
				return conflict("IDEMPOTENCY_KEY_REUSED")
			}
			status, response = storedStatus, storedResponse
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM session_requests WHERE user_id=$1 AND request_key=$2`, uid(c), key); err != nil {
			return err
		}
		r, previous, err := s.loadTarget(c, tx, input, sid, true)
		if err != nil {
			return err
		}
		if sid == "" {
			var existing string
			err = tx.QueryRow(ctx, `SELECT id::text FROM daily_sessions WHERE user_id=$1 AND work_date=$2 AND deleted_at IS NULL`, uid(c), r.Date).Scan(&existing)
			if err == nil {
				return conflict("SESSION_DATE_EXISTS", gin.H{"session_id": existing})
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		var claims previewClaims
		if verifySigned(token, s.config.PreviewKey, &claims) != nil {
			return stale("INVALID_TOKEN")
		}
		if claims.User != uid(c) || claims.Target != sid {
			return stale("TARGET_CHANGED")
		}
		if claims.Expires <= s.now().Unix() {
			return stale("EXPIRED")
		}
		if claims.Version != r.Version {
			return stale("VERSION_CHANGED")
		}
		if claims.InputHash != digest(input) || claims.MergedHash != mergedHash(r.Workday) {
			return stale("INPUT_CHANGED")
		}
		if claims.Efficiency != normalizedDecimal(r.efficiency) || claims.Price != normalizedDecimal(r.price) || claims.Reserve != normalizedDecimal(r.reserve) {
			return stale("COST_CHANGED")
		}
		if err = checkPlatforms(ctx, tx, r.Workday, previous); err != nil {
			return err
		}
		r.Summary, err = summary(r.Workday, r.efficiency, r.price, r.reserve)
		if err != nil {
			return err
		}
		if sid == "" {
			err = tx.QueryRow(ctx, `INSERT INTO daily_sessions(user_id,vehicle_id,work_date,distance_km,working_hours,note,snapshot_km_per_liter,snapshot_fuel_price,snapshot_maintenance_per_km) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text,version`, uid(c), r.Vehicle, r.Date, r.Distance, r.Hours, r.Note, r.efficiency, r.price, r.reserve).Scan(&r.ID, &r.Version)
		} else {
			if r.Version >= maxVersion {
				return conflict("PREVIEW_STALE", gin.H{"reason": "VERSION_CHANGED"})
			}
			err = tx.QueryRow(ctx, `UPDATE daily_sessions SET distance_km=$3,working_hours=$4,note=$5,version=version+1,updated_at=now() WHERE id=$1 AND user_id=$2 RETURNING version`, sid, uid(c), r.Distance, r.Hours, r.Note).Scan(&r.Version)
		}
		if err != nil {
			return err
		}
		// Full merged lists are saved; omitted lists retain the read-and-locked values.
		if _, err = tx.Exec(ctx, `DELETE FROM income_entries WHERE session_id=$1`, r.ID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM expense_entries WHERE session_id=$1`, r.ID); err != nil {
			return err
		}
		for _, i := range r.Incomes {
			if _, err = tx.Exec(ctx, `INSERT INTO income_entries(session_id,platform_id,gross_income) SELECT $1,id,$3 FROM platforms WHERE code=$2`, r.ID, i.Platform, i.Amount); err != nil {
				return err
			}
		}
		for _, x := range r.Expenses {
			if _, err = tx.Exec(ctx, `INSERT INTO expense_entries(session_id,category,amount,note) VALUES($1,$2,$3,$4)`, r.ID, x.Category, x.Amount, x.Note); err != nil {
				return err
			}
		}
		response, err = json.Marshal(gin.H{"data": confirmation{r.ID, r.Version, r.Summary}})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO session_requests(user_id,request_key,request_fingerprint,response_status,response_body,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, uid(c), key, []byte(fingerprint), status, response, s.now().Add(24*time.Hour))
		return err
	})
	if pgCode(e, "23505") {
		return conflict("SESSION_DATE_EXISTS")
	}
	if e != nil {
		return e
	}
	c.Data(status, "application/json; charset=utf-8", response)
	return nil
}
func (s *Server) getSession(c *gin.Context) error {
	sid, e := id(c, "session_id")
	if e != nil {
		return e
	}
	tx, e := s.db.BeginTx(c.Request.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Request.Context())
	r, e := readRecord(c.Request.Context(), tx, uid(c), sid, false)
	if e != nil {
		return e
	}
	if e = tx.Commit(c.Request.Context()); e != nil {
		return e
	}
	ok(c, 200, r.SavedDay)
	return nil
}
func (s *Server) deleteSession(c *gin.Context) error {
	sid, e := id(c, "session_id")
	if e != nil {
		return e
	}
	tag, e := s.db.Exec(c.Request.Context(), `UPDATE daily_sessions SET deleted_at=now(),version=version+1 WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, sid, uid(c))
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return missing()
	}
	c.Status(http.StatusNoContent)
	return nil
}
func (s *Server) queryDate(c *gin.Context, name string, required bool) (string, error) {
	v := c.Query(name)
	if v == "" && !required {
		return "", nil
	}
	e := s.validateDay(Workday{Date: v, Vehicle: "00000000-0000-0000-0000-000000000000"})
	if e != nil {
		return "", bad(name)
	}
	return v, nil
}
func (s *Server) dailyReport(c *gin.Context) error {
	date, e := s.queryDate(c, "date", true)
	if e != nil {
		return e
	}
	tx, e := s.db.BeginTx(c.Request.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Request.Context())
	var sid string
	e = tx.QueryRow(c.Request.Context(), `SELECT id::text FROM daily_sessions WHERE user_id=$1 AND work_date=$2 AND deleted_at IS NULL`, uid(c), date).Scan(&sid)
	if errors.Is(e, pgx.ErrNoRows) {
		ok(c, 200, nil)
		return nil
	}
	if e != nil {
		return e
	}
	r, e := readRecord(c.Request.Context(), tx, uid(c), sid, false)
	if e != nil {
		return e
	}
	if e = tx.Commit(c.Request.Context()); e != nil {
		return e
	}
	ok(c, 200, r.Summary)
	return nil
}
func pageParam(c *gin.Context, name string, def, max int) (int, error) {
	v := c.Query(name)
	if v == "" {
		return def, nil
	}
	n, e := strconv.Atoi(v)
	if e != nil || n < 1 || n > max {
		return 0, bad(name)
	}
	return n, nil
}
func (s *Server) history(c *gin.Context) error {
	from, e := s.queryDate(c, "from", false)
	if e != nil {
		return e
	}
	to, e := s.queryDate(c, "to", false)
	if e != nil {
		return e
	}
	if from != "" && to != "" && from > to {
		return bad("from")
	}
	page, e := pageParam(c, "page", 1, 2147483647)
	if e != nil {
		return e
	}
	size, e := pageParam(c, "page_size", 20, 100)
	if e != nil {
		return e
	}
	if from == "" {
		from = "0001-01-01"
	}
	if to == "" {
		to = "9999-12-31"
	}
	tx, e := s.db.BeginTx(c.Request.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Request.Context())
	rows, e := tx.Query(c.Request.Context(), `SELECT id::text FROM daily_sessions WHERE user_id=$1 AND work_date BETWEEN $2 AND $3 AND deleted_at IS NULL ORDER BY work_date DESC,id DESC LIMIT $4 OFFSET $5`, uid(c), from, to, size+1, int64(page-1)*int64(size))
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var sid string
		if e = rows.Scan(&sid); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, sid)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	more := len(ids) > size
	if more {
		ids = ids[:size]
	}
	values := []SavedDay{}
	for _, sid := range ids {
		r, err := readRecord(c.Request.Context(), tx, uid(c), sid, false)
		if err != nil {
			return err
		}
		values = append(values, r.SavedDay)
	}
	if e = tx.Commit(c.Request.Context()); e != nil {
		return e
	}
	c.JSON(200, gin.H{"data": values, "meta": gin.H{"page": page, "page_size": size, "has_more": more}})
	return nil
}
