package pilot

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Profile struct {
	ID          string   `json:"id"`
	DisplayName *string  `json:"display_name"`
	Timezone    string   `json:"timezone"`
	Preferred   []string `json:"preferred_platform_codes"`
	Onboarded   bool     `json:"onboarding_completed"`
}

func (s *Server) profile(ctx context.Context, q queryer, user string) (Profile, error) {
	p := Profile{Preferred: []string{}}
	e := q.QueryRow(ctx, `SELECT id::text,display_name,timezone,EXISTS(SELECT FROM vehicles WHERE user_id=users.id AND deleted_at IS NULL) AND EXISTS(SELECT FROM user_platforms WHERE user_id=users.id) FROM users WHERE id=$1 AND deleted_at IS NULL`, user).Scan(&p.ID, &p.DisplayName, &p.Timezone, &p.Onboarded)
	if e != nil {
		return p, e
	}
	rows, e := q.Query(ctx, `SELECT p.code FROM user_platforms u JOIN platforms p ON p.id=u.platform_id WHERE u.user_id=$1 ORDER BY p.code`, user)
	if e != nil {
		return p, e
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if e = rows.Scan(&code); e != nil {
			return p, e
		}
		p.Preferred = append(p.Preferred, code)
	}
	return p, rows.Err()
}
func (s *Server) getProfile(c *gin.Context) error {
	p, e := s.profile(c.Request.Context(), s.db, uid(c))
	if e != nil {
		return e
	}
	ok(c, 200, p)
	return nil
}
func (s *Server) putProfile(c *gin.Context) error {
	m, e := body(c, "display_name timezone preferred_platform_codes", "display_name timezone preferred_platform_codes")
	if e != nil {
		return e
	}
	name, e := str(m, "display_name")
	if e != nil || !length(name, 120) {
		return bad("display_name")
	}
	zone, e := str(m, "timezone")
	if e != nil || zone != "Asia/Bangkok" {
		return bad("timezone")
	}
	codes, found := m["preferred_platform_codes"].([]any)
	if !found || len(codes) > 4 {
		return bad("preferred_platform_codes")
	}
	seen := map[string]bool{}
	e = s.transaction(c, func(tx pgx.Tx) error {
		if _, err := tx.Exec(c.Request.Context(), `UPDATE users SET display_name=$2,updated_at=now() WHERE id=$1`, uid(c), name); err != nil {
			return err
		}
		if _, err := tx.Exec(c.Request.Context(), `DELETE FROM user_platforms WHERE user_id=$1`, uid(c)); err != nil {
			return err
		}
		for _, value := range codes {
			code, valid := value.(string)
			if !valid || seen[code] {
				return bad("preferred_platform_codes")
			}
			seen[code] = true
			var platform string
			if err := tx.QueryRow(c.Request.Context(), `SELECT id::text FROM platforms WHERE code=$1 AND is_active FOR SHARE`, code).Scan(&platform); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return bad("preferred_platform_codes")
				}
				return err
			}
			if _, err := tx.Exec(c.Request.Context(), `INSERT INTO user_platforms(user_id,platform_id) VALUES($1,$2)`, uid(c), platform); err != nil {
				return err
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	return s.getProfile(c)
}
func (s *Server) platforms(c *gin.Context) error {
	rows, e := s.db.Query(c.Request.Context(), `SELECT code,name FROM platforms WHERE is_active ORDER BY code`)
	if e != nil {
		return e
	}
	defer rows.Close()
	values := []gin.H{}
	for rows.Next() {
		var code, name string
		if e = rows.Scan(&code, &name); e != nil {
			return e
		}
		values = append(values, gin.H{"code": code, "name": name})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	ok(c, 200, values)
	return nil
}

type Vehicle struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Efficiency string `json:"km_per_liter"`
	Price      string `json:"fuel_price_per_liter"`
	Reserve    string `json:"maintenance_reserve_per_km"`
	Default    bool   `json:"is_default"`
}

const vehicleCols = `id::text,name,km_per_liter::text,fuel_price_per_liter::text,maintenance_reserve_per_km::text,is_default`

func readVehicle(ctx context.Context, q queryer, user, vehicle string, lock bool) (Vehicle, error) {
	var v Vehicle
	sql := `SELECT ` + vehicleCols + ` FROM vehicles WHERE user_id=$1 AND deleted_at IS NULL`
	args := []any{user}
	if vehicle != "" {
		sql += ` AND id=$2`
		args = append(args, vehicle)
	}
	if lock {
		sql += ` FOR UPDATE`
	}
	e := q.QueryRow(ctx, sql, args...).Scan(&v.ID, &v.Name, &v.Efficiency, &v.Price, &v.Reserve, &v.Default)
	if errors.Is(e, pgx.ErrNoRows) {
		e = missing()
	}
	return v, e
}
func (s *Server) vehicles(c *gin.Context) error {
	v, e := readVehicle(c.Request.Context(), s.db, uid(c), "", false)
	values := []Vehicle{}
	var a *apiError
	if errors.As(e, &a) && a.status == 404 {
		e = nil
	} else if e == nil {
		values = append(values, v)
	}
	if e != nil {
		return e
	}
	ok(c, 200, values)
	return nil
}

var decimalRE = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

func decimal(value string, whole, scale int, positive bool) bool {
	if len(value) > 32 || !decimalRE.MatchString(value) {
		return false
	}
	parts := strings.Split(value, ".")
	if len(strings.TrimLeft(parts[0], "0")) > whole || (len(parts) == 2 && len(parts[1]) > scale) {
		return false
	}
	return !positive || strings.Trim(value, "0.") != ""
}
func vehicleInput(m map[string]any, v Vehicle) (Vehicle, error) {
	for k, x := range m {
		switch k {
		case "is_default":
			b, ok := x.(bool)
			if !ok || !b {
				return v, bad(k)
			}
		case "name":
			name, ok := x.(string)
			if !ok || name == "" || !length(name, 100) {
				return v, bad(k)
			}
			v.Name = name
		case "km_per_liter", "fuel_price_per_liter", "maintenance_reserve_per_km":
			value, ok := x.(string)
			whole, scale := 5, 3
			if k == "fuel_price_per_liter" {
				whole, scale = 6, 2
			}
			if !ok || !decimal(value, whole, scale, k == "km_per_liter") {
				return v, bad(k)
			}
			if k == "km_per_liter" {
				v.Efficiency = value
			} else if k == "fuel_price_per_liter" {
				v.Price = value
			} else {
				v.Reserve = value
			}
		}
	}
	v.Default = true
	return v, nil
}

const vehicleKeys = "name km_per_liter fuel_price_per_liter maintenance_reserve_per_km is_default"

func (s *Server) createVehicle(c *gin.Context) error {
	m, e := body(c, vehicleKeys, "name km_per_liter fuel_price_per_liter maintenance_reserve_per_km")
	if e != nil {
		return e
	}
	v, e := vehicleInput(m, Vehicle{})
	if e != nil {
		return e
	}
	e = s.db.QueryRow(c.Request.Context(), `INSERT INTO vehicles(user_id,name,km_per_liter,fuel_price_per_liter,maintenance_reserve_per_km) VALUES($1,$2,$3,$4,$5) RETURNING `+vehicleCols, uid(c), v.Name, v.Efficiency, v.Price, v.Reserve).Scan(&v.ID, &v.Name, &v.Efficiency, &v.Price, &v.Reserve, &v.Default)
	if pgCode(e, "23505") {
		return conflict("VEHICLE_EXISTS")
	}
	if e != nil {
		return e
	}
	ok(c, 201, v)
	return nil
}
func (s *Server) patchVehicle(c *gin.Context) error {
	vid, e := id(c, "vehicle_id")
	if e != nil {
		return e
	}
	m, e := body(c, vehicleKeys, "")
	if e != nil {
		return e
	}
	if len(m) == 0 {
		return bad("body")
	}
	var v Vehicle
	e = s.transaction(c, func(tx pgx.Tx) error {
		var err error
		v, err = readVehicle(c.Request.Context(), tx, uid(c), vid, true)
		if err != nil {
			return err
		}
		v, err = vehicleInput(m, v)
		if err != nil {
			return err
		}
		return tx.QueryRow(c.Request.Context(), `UPDATE vehicles SET name=$3,km_per_liter=$4,fuel_price_per_liter=$5,maintenance_reserve_per_km=$6,updated_at=now() WHERE id=$1 AND user_id=$2 RETURNING `+vehicleCols, vid, uid(c), v.Name, v.Efficiency, v.Price, v.Reserve).Scan(&v.ID, &v.Name, &v.Efficiency, &v.Price, &v.Reserve, &v.Default)
	})
	if e != nil {
		return e
	}
	ok(c, 200, v)
	return nil
}
