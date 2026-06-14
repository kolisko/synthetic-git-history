package schedule

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kolisko/synthetic-git-history/internal/config"
)

func TestScheduleIsDeterministicWithSeed(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}

	first := Build(cfg)
	second := Build(cfg)

	if len(first) == 0 {
		t.Fatal("expected planned commits")
	}
	if len(first) != len(second) {
		t.Fatalf("commit count differs: %d != %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("schedule differs at index %d", i)
		}
	}
}

func TestWeekendsCanBeSkipped(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Range.SkipWeekends = true
	cfg.Volume.ActiveDayProb = 1

	specs := Build(cfg)
	if len(specs) == 0 {
		t.Fatal("expected planned commits")
	}
	for _, spec := range specs {
		if spec.Timestamp.Weekday() == time.Saturday || spec.Timestamp.Weekday() == time.Sunday {
			t.Fatalf("weekend commit was planned: %s", spec.Timestamp)
		}
	}
}
