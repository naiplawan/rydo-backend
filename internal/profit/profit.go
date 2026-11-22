// Package profit calculates estimated workday costs from decimal inputs.
package profit

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type Workday struct {
	Distance, Hours, FuelEfficiency, FuelPrice, MaintenanceRate string
	Income, Expenses                                            []string
}

type Summary struct {
	GrossIncome, Fuel, Maintenance, OtherExpenses, TotalCost, NetProfit string
	PerHour, PerKilometer                                               *string
}

func Calculate(day Workday) (Summary, error) {
	if len(day.Income) == 0 {
		return Summary{}, errors.New("at least one income entry is required")
	}

	fields := []struct {
		name, value     string
		integers, scale int
	}{
		{"distance_km", day.Distance, 8, 2},
		{"working_hours", day.Hours, 6, 2},
		{"km_per_liter", day.FuelEfficiency, 5, 3},
		{"fuel_price_per_liter", day.FuelPrice, 6, 2},
		{"maintenance_reserve_per_km", day.MaintenanceRate, 5, 3},
	}
	parsed := make([]*big.Rat, len(fields))
	for i, field := range fields {
		value, err := parseDecimal(field.value, field.integers, field.scale)
		if err != nil {
			return Summary{}, fmt.Errorf("%s: %w", field.name, err)
		}
		parsed[i] = value
	}
	distance, hours, efficiency, price, reserve := parsed[0], parsed[1], parsed[2], parsed[3], parsed[4]
	if efficiency.Sign() == 0 {
		return Summary{}, errors.New("km_per_liter must be positive")
	}
	if hours.Cmp(big.NewRat(24, 1)) > 0 {
		return Summary{}, errors.New("working_hours must be at most 24")
	}

	income, err := sumMoney(day.Income)
	if err != nil {
		return Summary{}, fmt.Errorf("income: %w", err)
	}
	expenses, err := sumMoney(day.Expenses)
	if err != nil {
		return Summary{}, fmt.Errorf("expenses: %w", err)
	}

	// Keep each calculation exact until its specified rounding boundary.
	fuel := new(big.Rat).Quo(distance, efficiency)
	fuel.Mul(fuel, price)
	fuelAmount := fuel.FloatString(2)
	maintenanceAmount := new(big.Rat).Mul(distance, reserve).FloatString(2)
	fuel.SetString(fuelAmount)
	maintenance, _ := new(big.Rat).SetString(maintenanceAmount)
	cost := new(big.Rat).Add(fuel, maintenance)
	cost.Add(cost, expenses)
	net := new(big.Rat).Sub(income, cost)

	return Summary{
		GrossIncome:   income.FloatString(2),
		Fuel:          fuelAmount,
		Maintenance:   maintenanceAmount,
		OtherExpenses: expenses.FloatString(2),
		TotalCost:     cost.FloatString(2),
		NetProfit:     net.FloatString(2),
		PerHour:       rate(net, hours),
		PerKilometer:  rate(net, distance),
	}, nil
}

func parseDecimal(value string, integers, scale int) (*big.Rat, error) {
	whole, fractional, hasPoint := strings.Cut(value, ".")
	if whole == "" || len(strings.TrimLeft(whole, "0")) > integers ||
		(hasPoint && (fractional == "" || len(fractional) > scale)) {
		return nil, errors.New("decimal exceeds the allowed precision or scale")
	}
	for _, digit := range whole + fractional {
		if digit < '0' || digit > '9' {
			return nil, errors.New("expected a nonnegative decimal string")
		}
	}
	result, _ := new(big.Rat).SetString(value)
	return result, nil
}

func sumMoney(amounts []string) (*big.Rat, error) {
	total := new(big.Rat)
	for i, amount := range amounts {
		value, err := parseDecimal(amount, 10, 2)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i+1, err)
		}
		total.Add(total, value)
	}
	return total, nil
}

func rate(net, denominator *big.Rat) *string {
	if denominator.Sign() == 0 {
		return nil
	}
	amount := new(big.Rat).Quo(net, denominator).FloatString(2)
	if amount == "-0.00" {
		amount = "0.00"
	}
	return &amount
}
