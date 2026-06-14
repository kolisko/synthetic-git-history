package schedule

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kolisko/synthetic-git-history/internal/config"
)

type CommitSpec struct {
	Timestamp  time.Time
	Timezone   string
	Message    string
	Line       string
	Sequence   int
	DayIndex   int
	DailyTotal int
}

func (spec CommitSpec) GitDate() string {
	return spec.Timestamp.Format("2006-01-02T15:04:05") + " " + spec.Timezone
}

func Build(cfg config.Config) []CommitSpec {
	seed := time.Now().UnixNano()
	if cfg.Seed != nil {
		seed = *cfg.Seed
	}
	rng := rand.New(rand.NewSource(seed))

	var specs []CommitSpec
	sequence := 1

	for day := cfg.Range.Start; !day.After(cfg.Range.End); day = day.AddDate(0, 0, 1) {
		if cfg.Range.SkipWeekends && isWeekend(day) {
			continue
		}

		probability := cfg.Volume.ActiveDayProb
		if isWeekend(day) {
			probability *= cfg.Volume.WeekendMultiplier
		}
		if rng.Float64() > probability {
			continue
		}

		total := randomInt(rng, cfg.Volume.MinCommitsPerDay, cfg.Volume.MaxCommitsPerDay)
		timestamps := randomTimesForDay(rng, day, cfg.Time.Start, cfg.Time.End, total)
		sort.Slice(timestamps, func(i, j int) bool {
			return timestamps[i].Before(timestamps[j])
		})

		for index, timestamp := range timestamps {
			dayIndex := index + 1
			values := templateValues(timestamp, sequence, dayIndex, total)
			messageTemplate := cfg.Content.MessageTemplates[randomInt(rng, 0, len(cfg.Content.MessageTemplates)-1)]
			specs = append(specs, CommitSpec{
				Timestamp:  timestamp,
				Timezone:   cfg.Range.Timezone,
				Message:    render(messageTemplate, values),
				Line:       render(cfg.Content.LineTemplate, values),
				Sequence:   sequence,
				DayIndex:   dayIndex,
				DailyTotal: total,
			})
			sequence++
		}
	}

	return specs
}

func randomTimesForDay(rng *rand.Rand, day time.Time, start, end time.Duration, count int) []time.Time {
	spanSeconds := int((end - start) / time.Second)
	timestamps := make([]time.Time, 0, count)
	for i := 0; i < count; i++ {
		offset := time.Duration(rng.Intn(spanSeconds+1)) * time.Second
		timestamps = append(timestamps, day.Add(start).Add(offset))
	}
	return timestamps
}

func templateValues(timestamp time.Time, sequence, index, dailyTotal int) map[string]string {
	return map[string]string{
		"date":        timestamp.Format("2006-01-02"),
		"time":        timestamp.Format("15:04:05"),
		"index":       strconv.Itoa(index),
		"daily_total": strconv.Itoa(dailyTotal),
		"sequence":    strconv.Itoa(sequence),
	}
}

func render(template string, values map[string]string) string {
	result := template
	for key, value := range values {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}

func randomInt(rng *rand.Rand, min, max int) int {
	if min == max {
		return min
	}
	return min + rng.Intn(max-min+1)
}

func isWeekend(day time.Time) bool {
	return day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
}
