package persistence

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSequenceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	engine := NewGormSequenceEngine(db)
	if err := engine.MigrateSequenceTable(); err != nil {
		t.Fatalf("failed to migrate sequences table: %v", err)
	}

	return db
}

func TestSequenceMigrateTable(t *testing.T) {
	db := newSequenceTestDB(t)
	if !db.Migrator().HasTable("sequences") {
		t.Fatal("expected sequences table to exist")
	}
}

func TestSequenceNextValueFirstCallReturnsOne(t *testing.T) {
	engine := NewGormSequenceEngine(newSequenceTestDB(t))

	value, err := engine.NextValue("order", "number", "order:number", 1)
	if err != nil {
		t.Fatalf("NextValue error: %v", err)
	}
	if value != 1 {
		t.Fatalf("expected first value 1, got %d", value)
	}
}

func TestSequenceNextValueSecondCallReturnsTwo(t *testing.T) {
	engine := NewGormSequenceEngine(newSequenceTestDB(t))

	if _, err := engine.NextValue("order", "number", "order:number", 1); err != nil {
		t.Fatalf("first NextValue error: %v", err)
	}

	value, err := engine.NextValue("order", "number", "order:number", 1)
	if err != nil {
		t.Fatalf("second NextValue error: %v", err)
	}
	if value != 2 {
		t.Fatalf("expected second value 2, got %d", value)
	}
}

func TestSequenceNextValueDifferentKeysAreIndependent(t *testing.T) {
	engine := NewGormSequenceEngine(newSequenceTestDB(t))

	firstA, err := engine.NextValue("order", "number", "order:number:a", 1)
	if err != nil {
		t.Fatalf("first key error: %v", err)
	}
	firstB, err := engine.NextValue("order", "number", "order:number:b", 1)
	if err != nil {
		t.Fatalf("second key error: %v", err)
	}
	secondA, err := engine.NextValue("order", "number", "order:number:a", 1)
	if err != nil {
		t.Fatalf("first key second call error: %v", err)
	}

	if firstA != 1 || firstB != 1 || secondA != 2 {
		t.Fatalf("expected independent sequences [1,1,2], got [%d,%d,%d]", firstA, firstB, secondA)
	}
}

func TestSequenceNextValueWithStepTwo(t *testing.T) {
	engine := NewGormSequenceEngine(newSequenceTestDB(t))

	first, err := engine.NextValue("invoice", "number", "invoice:number", 2)
	if err != nil {
		t.Fatalf("first NextValue error: %v", err)
	}
	second, err := engine.NextValue("invoice", "number", "invoice:number", 2)
	if err != nil {
		t.Fatalf("second NextValue error: %v", err)
	}

	if first != 1 || second != 3 {
		t.Fatalf("expected stepped values [1,3], got [%d,%d]", first, second)
	}
}

func TestSequenceNextValueConcurrentReturnsUniqueValues(t *testing.T) {
	engine := NewGormSequenceEngine(newSequenceTestDB(t))

	const count = 10
	results := make([]int64, count)
	errs := make([]error, count)
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errs[index] = engine.NextValue("shipment", "number", "shipment:number", 1)
		}(i)
	}

	wg.Wait()

	seen := make(map[int64]bool, count)
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d NextValue error: %v", i, err)
		}
		if seen[results[i]] {
			t.Fatalf("duplicate sequence value detected: %d", results[i])
		}
		seen[results[i]] = true
	}

	sorted := append([]int64(nil), results...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for i, value := range sorted {
		expected := int64(i + 1)
		if value != expected {
			t.Fatalf("expected sorted value %d at position %d, got %d", expected, i, value)
		}
	}
}

func TestSequenceBuildSequenceKey(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		reset  string
		want   string
	}{
		{name: "empty uses base key", reset: "", want: "order:number"},
		{name: "never uses base key", reset: "never", want: "order:number"},
		{name: "yearly", reset: "yearly", want: now.Format("2006")},
		{name: "monthly", reset: "monthly", want: now.Format("2006-01")},
		{name: "daily", reset: "daily", want: now.Format("2006-01-02")},
		{name: "hourly", reset: "hourly", want: now.Format("2006-01-02T15")},
		{name: "minutely", reset: "minutely", want: now.Format("2006-01-02T15:04")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSequenceKey("order", "number", tt.reset)
			if tt.reset == "" || tt.reset == "never" {
				if got != tt.want {
					t.Fatalf("expected %s, got %s", tt.want, got)
				}
				return
			}

			expected := "order:number:" + tt.want
			if got != expected {
				t.Fatalf("expected %s, got %s", expected, got)
			}
		})
	}
}
