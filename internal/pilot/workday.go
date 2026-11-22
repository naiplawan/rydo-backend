package pilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"rydo-backend/internal/profit"
)

type Income struct {
	Platform string `json:"platform_code"`
	Amount   string `json:"gross_income"`
}
type Expense struct {
	Category string `json:"category"`
	Amount   string `json:"amount"`
	Note     string `json:"note"`
}
type Workday struct {
	Date     string    `json:"work_date"`
	Vehicle  string    `json:"vehicle_id"`
	Distance string    `json:"distance_km"`
	Hours    string    `json:"working_hours"`
	Incomes  []Income  `json:"incomes"`
	Expenses []Expense `json:"expenses"`
	Note     string    `json:"note"`
}
type Costs struct {
	Fuel    string `json:"fuel_estimated"`
	Reserve string `json:"maintenance_reserve"`
	Other   string `json:"other_actual"`
}
type Summary struct {
	Date     string  `json:"date"`
	Currency string  `json:"currency"`
	Gross    string  `json:"gross_income"`
	Costs    Costs   `json:"costs"`
	Total    string  `json:"total_cost"`
	Net      string  `json:"estimated_net_profit"`
	Distance string  `json:"distance_km"`
	Hours    string  `json:"working_hours"`
	PerKM    *string `json:"estimated_net_per_km"`
	PerHour  *string `json:"estimated_net_per_hour"`
}
type SavedDay struct {
	Workday
	ID      string  `json:"id"`
	Version int64   `json:"version"`
	Summary Summary `json:"summary"`
}
type record struct {
	SavedDay
	efficiency, price, reserve string
}
type confirmation struct {
	ID      string  `json:"id"`
	Version int64   `json:"version"`
	Summary Summary `json:"summary"`
}

const maxVersion int64 = 9007199254740991
const createFields = "work_date vehicle_id distance_km working_hours incomes expenses note"
const editFields = "distance_km working_hours incomes expenses note"
const createRequired = "work_date vehicle_id distance_km working_hours incomes"

func normalizedDecimal(v string) string {
	p := strings.Split(v, ".")
	whole := strings.TrimLeft(p[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(p) == 1 {
		return whole
	}
	fraction := strings.TrimRight(p[1], "0")
	if fraction == "" {
		return whole
	}
	return whole + "." + fraction
}
func noNull(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case map[string]any:
		for _, v := range x {
			if !noNull(v) {
				return false
			}
		}
	case []any:
		for _, v := range x {
			if !noNull(v) {
				return false
			}
		}
	}
	return true
}
func canonicalMap(m map[string]any, create bool) ([]byte, error) {
	if !noNull(m) {
		return nil, bad("body")
	}
	var walk func(any) error
	walk = func(v any) error {
		switch x := v.(type) {
		case map[string]any:
			for k, value := range x {
				switch k {
				case "distance_km", "working_hours", "gross_income", "amount":
					text, ok := value.(string)
					if !ok || len(text) > 32 || !decimalRE.MatchString(text) {
						return bad(k)
					}
					x[k] = normalizedDecimal(text)
				case "vehicle_id":
					text, ok := value.(string)
					if !ok || !uuidRE.MatchString(text) {
						return bad(k)
					}
					x[k] = strings.ToLower(text)
				default:
					if e := walk(value); e != nil {
						return e
					}
				}
			}
		case []any:
			for _, v := range x {
				if e := walk(v); e != nil {
					return e
				}
			}
		}
		return nil
	}
	// Precision/scale must be checked before canonicalization can remove excess zeroes.
	if e := validateProvided(m); e != nil {
		return nil, e
	}
	if e := walk(m); e != nil {
		return nil, e
	}
	if create {
		if _, ok := m["expenses"]; !ok {
			m["expenses"] = []any{}
		}
		if _, ok := m["note"]; !ok {
			m["note"] = ""
		}
	}
	if values, ok := m["expenses"].([]any); ok {
		for _, value := range values {
			row := value.(map[string]any)
			if _, ok := row["note"]; !ok {
				row["note"] = ""
			}
		}
	}
	return json.Marshal(m)
}
func validateProvided(m map[string]any) error {
	for k, value := range m {
		switch k {
		case "work_date":
			v, ok := value.(string)
			if !ok || len(v) != 10 {
				return bad(k)
			}
		case "distance_km", "working_hours":
			v, ok := value.(string)
			if !ok || !decimal(v, 8, 2, false) {
				return bad(k)
			}
			if k == "working_hours" {
				n, _ := new(big.Rat).SetString(v)
				if n.Cmp(big.NewRat(24, 1)) > 0 {
					return bad(k)
				}
			}
		case "note":
			v, ok := value.(string)
			if !ok || !length(v, 1000) {
				return bad(k)
			}
		case "incomes":
			values, ok := value.([]any)
			if !ok || len(values) < 1 || len(values) > 4 {
				return bad(k)
			}
			seen := map[string]bool{}
			for _, value := range values {
				row, ok := value.(map[string]any)
				if !ok {
					return bad(k)
				}
				if e := keys(row, "platform_code gross_income", "platform_code gross_income"); e != nil {
					return e
				}
				code, ok := row["platform_code"].(string)
				if !ok || !platformCode(code) || seen[code] {
					return bad("incomes")
				}
				seen[code] = true
				amount, ok := row["gross_income"].(string)
				if !ok || !decimal(amount, 10, 2, false) {
					return bad("incomes")
				}
			}
		case "expenses":
			values, ok := value.([]any)
			if !ok || len(values) > 100 {
				return bad(k)
			}
			for _, value := range values {
				row, ok := value.(map[string]any)
				if !ok {
					return bad(k)
				}
				if e := keys(row, "category amount note", "category amount"); e != nil {
					return e
				}
				category, ok := row["category"].(string)
				if !ok || (category != "parking" && category != "toll" && category != "phone" && category != "other") {
					return bad("expenses")
				}
				amount, ok := row["amount"].(string)
				if !ok || !decimal(amount, 10, 2, false) {
					return bad("expenses")
				}
				if note, ok := row["note"]; ok {
					v, valid := note.(string)
					if !valid || !length(v, 1000) {
						return bad("expenses.note")
					}
				}
			}
		}
	}
	return nil
}
func platformCode(v string) bool {
	return v == "grab" || v == "lineman" || v == "shopeefood" || v == "other"
}
func (s *Server) validateDay(d Workday) error {
	date, e := time.Parse("2006-01-02", d.Date)
	if e != nil || date.Year() < 1 || d.Date > s.now().In(time.FixedZone("Asia/Bangkok", 7*3600)).Format("2006-01-02") {
		return bad("work_date")
	}
	if !uuidRE.MatchString(d.Vehicle) {
		return bad("vehicle_id")
	}
	return nil
}
func summary(d Workday, efficiency, price, reserve string) (Summary, error) {
	income, expense := []string{}, []string{}
	for _, i := range d.Incomes {
		income = append(income, i.Amount)
	}
	for _, e := range d.Expenses {
		expense = append(expense, e.Amount)
	}
	v, e := profit.Calculate(profit.Workday{Distance: d.Distance, Hours: d.Hours, FuelEfficiency: efficiency, FuelPrice: price, MaintenanceRate: reserve, Income: income, Expenses: expense})
	if e != nil {
		return Summary{}, e
	}
	distance, _ := new(big.Rat).SetString(d.Distance)
	hours, _ := new(big.Rat).SetString(d.Hours)
	return Summary{d.Date, "THB", v.GrossIncome, Costs{v.Fuel, v.Maintenance, v.OtherExpenses}, v.TotalCost, v.NetProfit, distance.FloatString(2), hours.FloatString(2), v.PerKilometer, v.PerHour}, nil
}
func digest(v []byte) string { sum := sha256.Sum256(v); return hex.EncodeToString(sum[:]) }
func mergedInput(d Workday, patch []byte) (Workday, error) {
	raw, e := json.Marshal(d)
	if e != nil {
		return d, e
	}
	m := map[string]any{}
	if e = json.Unmarshal(raw, &m); e != nil {
		return d, e
	}
	p := map[string]any{}
	if e = json.Unmarshal(patch, &p); e != nil {
		return d, e
	}
	for k, v := range p {
		m[k] = v
	}
	raw, e = json.Marshal(m)
	if e != nil {
		return d, e
	}
	e = json.Unmarshal(raw, &d)
	return d, e
}
func mergedHash(d Workday) string {
	raw, _ := json.Marshal(d)
	m := map[string]any{}
	json.Unmarshal(raw, &m)
	norm, _ := canonicalMap(m, true)
	return digest(norm)
}

func readRecord(ctx context.Context, q queryer, user, sid string, lock bool) (record, error) {
	r := record{}
	sql := `SELECT id::text,vehicle_id::text,work_date::text,distance_km::text,working_hours::text,COALESCE(note,''),version,snapshot_km_per_liter::text,snapshot_fuel_price::text,snapshot_maintenance_per_km::text FROM daily_sessions WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`
	if lock {
		sql += ` FOR UPDATE`
	}
	e := q.QueryRow(ctx, sql, sid, user).Scan(&r.ID, &r.Vehicle, &r.Date, &r.Distance, &r.Hours, &r.Note, &r.Version, &r.efficiency, &r.price, &r.reserve)
	if errors.Is(e, pgx.ErrNoRows) {
		return r, missing()
	}
	if e != nil {
		return r, e
	}
	r.Incomes = []Income{}
	r.Expenses = []Expense{}
	rows, e := q.Query(ctx, `SELECT p.code,i.gross_income::text FROM income_entries i JOIN platforms p ON p.id=i.platform_id WHERE i.session_id=$1 AND i.deleted_at IS NULL ORDER BY i.created_at,i.id`, sid)
	if e != nil {
		return r, e
	}
	for rows.Next() {
		var i Income
		if e = rows.Scan(&i.Platform, &i.Amount); e != nil {
			rows.Close()
			return r, e
		}
		r.Incomes = append(r.Incomes, i)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return r, e
	}
	rows, e = q.Query(ctx, `SELECT category,amount::text,COALESCE(note,'') FROM expense_entries WHERE session_id=$1 AND deleted_at IS NULL ORDER BY created_at,id`, sid)
	if e != nil {
		return r, e
	}
	for rows.Next() {
		var x Expense
		if e = rows.Scan(&x.Category, &x.Amount, &x.Note); e != nil {
			rows.Close()
			return r, e
		}
		r.Expenses = append(r.Expenses, x)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return r, e
	}
	r.Summary, e = summary(r.Workday, r.efficiency, r.price, r.reserve)
	return r, e
}
func checkPlatforms(ctx context.Context, q queryer, d Workday, previous []Income) error {
	retained := map[string]bool{}
	for _, i := range previous {
		retained[i.Platform] = true
	}
	for _, i := range d.Incomes {
		var active bool
		e := q.QueryRow(ctx, `SELECT is_active FROM platforms WHERE code=$1 FOR SHARE`, i.Platform).Scan(&active)
		if errors.Is(e, pgx.ErrNoRows) {
			return bad("incomes")
		}
		if e != nil {
			return e
		}
		if !active && !retained[i.Platform] {
			return bad("incomes")
		}
	}
	return nil
}
