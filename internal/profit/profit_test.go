package profit

import "testing"

func TestCalculateDocumentedWorkday(t *testing.T) {
	got, err := Calculate(Workday{
		Distance: "100.00", Hours: "8.00", FuelEfficiency: "40.000",
		FuelPrice: "35.00", MaintenanceRate: "0.600",
		Income: []string{"500.00", "250.00", "100.00"}, Expenses: []string{"30.00"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.GrossIncome != "850.00" || got.Fuel != "87.50" || got.Maintenance != "60.00" ||
		got.OtherExpenses != "30.00" || got.TotalCost != "177.50" || got.NetProfit != "672.50" {
		t.Fatalf("wrong financial summary: %+v", got)
	}
	if got.PerHour == nil || *got.PerHour != "84.06" || got.PerKilometer == nil || *got.PerKilometer != "6.73" {
		t.Fatal("expected 84.06 per hour and 6.73 per km")
	}
}

func TestCalculateRoundsCostsBeforeSumming(t *testing.T) {
	got, err := Calculate(Workday{
		Distance: "1", Hours: "2", FuelEfficiency: "2", FuelPrice: "0.01",
		MaintenanceRate: "0.005", Expenses: []string{"13.43"}, Income: []string{"0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Fuel != "0.01" || got.Maintenance != "0.01" || got.NetProfit != "-13.45" {
		t.Fatalf("wrong financial summary: %+v", got)
	}
	if got.PerHour == nil || *got.PerHour != "-6.73" {
		t.Fatal("expected negative rounding tie to round away from zero")
	}
}

func TestCalculateZeroDenominators(t *testing.T) {
	got, err := Calculate(Workday{Distance: "0", Hours: "0", FuelEfficiency: "40", FuelPrice: "35", MaintenanceRate: "0", Income: []string{"1"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.PerHour != nil || got.PerKilometer != nil {
		t.Fatal("zero denominators must return nil rates")
	}
}

func TestCalculateRejectsInvalidDecimals(t *testing.T) {
	for _, input := range []string{"1/2", "0.001", "1e2", "-1", "100000000.00"} {
		if _, err := Calculate(Workday{Distance: input, Hours: "1", FuelEfficiency: "40", FuelPrice: "35", MaintenanceRate: "0", Income: []string{"1"}}); err == nil {
			t.Errorf("invalid distance %q was accepted", input)
		}
	}
}
