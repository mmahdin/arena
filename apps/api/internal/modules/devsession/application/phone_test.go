package application

import "testing"

func TestNormalizeSyntheticPhone(t *testing.T) {
	for _, input := range []string{"09000000001", "۰۹۰۰۰۰۰۰۰۰۱", "٠٩٠٠٠٠٠٠٠٠١", " 09000000001 "} {
		got, err := NormalizeSyntheticPhone(input)
		if err != nil || got != "09000000001" {
			t.Fatalf("synthetic format rejected: %q / %v", got, err)
		}
	}
	for _, input := range []string{"09123456789", "۰۹۱۲۳۴۵۶۷۸۹", "٠٩١٢٣٤٥٦٧٨٩"} {
		got, err := NormalizeSyntheticPhone(input)
		if err != nil || got != "09123456789" {
			t.Fatalf("valid mobile rejected: %q / %v", got, err)
		}
	}
	for _, input := range []string{"", "0900000000", "090000000011", "0900000a001", "0900000-001", "009000000001", "0900000\n001"} {
		if _, err := NormalizeSyntheticPhone(input); err == nil {
			t.Fatal("accepted invalid synthetic identifier")
		}
	}
}
