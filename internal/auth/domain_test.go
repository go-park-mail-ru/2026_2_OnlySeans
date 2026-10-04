package auth

import (
	"testing"
	"time"
)

func TestTSTZRangeValueAndScan(t *testing.T) {
	lower := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	upper := lower.Add(24 * time.Hour)
	want := TSTZRange{
		Lower:          &lower,
		Upper:          &upper,
		LowerInclusive: true,
	}

	value, err := want.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	var got TSTZRange
	if err := got.Scan(value); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if got.Lower == nil || !got.Lower.Equal(lower) {
		t.Errorf("lower bound = %v, want %v", got.Lower, lower)
	}
	if got.Upper == nil || !got.Upper.Equal(upper) {
		t.Errorf("upper bound = %v, want %v", got.Upper, upper)
	}
	if !got.LowerInclusive || got.UpperInclusive {
		t.Errorf("inclusivity = (%t, %t), want (true, false)", got.LowerInclusive, got.UpperInclusive)
	}
}

func TestTSTZRangeScanUnbounded(t *testing.T) {
	var got TSTZRange
	if err := got.Scan(`(,"2026-10-03T12:00:00Z"]`); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if got.Lower != nil || got.LowerInclusive {
		t.Errorf("lower bound = (%v, %t), want unbounded", got.Lower, got.LowerInclusive)
	}
	if got.Upper == nil || !got.UpperInclusive {
		t.Errorf("upper bound = (%v, %t), want an inclusive bound", got.Upper, got.UpperInclusive)
	}
}

func TestDomainConstructorsApplyDatabaseDefaults(t *testing.T) {
	if got := NewFilmFile(1, 2, "poster").SortOrder; got != 1 {
		t.Errorf("NewFilmFile().SortOrder = %d, want 1", got)
	}
	if got := NewSeason(1, 1).FilmType; got != "series" {
		t.Errorf("NewSeason().FilmType = %q, want series", got)
	}
	if got := NewFolder(1, "Favorites", "favorite").IsPrivate; !got {
		t.Error("NewFolder().IsPrivate = false, want true")
	}
}
