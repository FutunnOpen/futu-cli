package cmd

import (
	"testing"
	"time"
)

func TestDefaultOptionChainRange(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	start, end := defaultOptionChainRange(now)
	if start != "2026-09-21" {
		t.Fatalf("start=%s want 2026-09-21", start)
	}
	if end != "2026-10-21" {
		t.Fatalf("end=%s want 2026-10-21", end)
	}
}

func TestPickOptionChainRange(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	start, end := pickOptionChainRange(now, []string{"2026-11-12", "2026-12-10", "2027-03-11"}, 30)
	if start != "2026-11-12" || end != "2026-11-12" {
		t.Fatalf("jp monthly: start=%s end=%s want 2026-11-12/2026-11-12", start, end)
	}

	start, end = pickOptionChainRange(now, []string{"2026-10-09", "2026-10-16", "2026-10-29", "2026-11-27"}, 30)
	if start != "2026-10-09" || end != "2026-10-29" {
		t.Fatalf("hk weekly: start=%s end=%s want 2026-10-09/2026-10-29", start, end)
	}

	start, end = pickOptionChainRange(now, []string{"2026-09-01", "2026-10-16"}, 30)
	if start != "2026-10-16" || end != "2026-10-16" {
		t.Fatalf("ignore past: start=%s end=%s want 2026-10-16/2026-10-16", start, end)
	}

	start, end = pickOptionChainRange(now, nil, 30)
	if start != "2026-10-09" || end != "2026-11-08" {
		t.Fatalf("empty fallback: start=%s end=%s want 2026-10-09/2026-11-08", start, end)
	}
}

func TestIndexOptionTypeBySymbol(t *testing.T) {
	if indexOptionTypeBySymbol["HK.800000"] != 1 {
		t.Fatal("HSI should map to 1")
	}
	if _, ok := indexOptionTypeBySymbol["HK.00700"]; ok {
		t.Fatal("equity should not be mapped")
	}
}
