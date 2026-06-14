package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Seed       *int64           `json:"seed"`
	Repository RepositoryConfig `json:"repository"`
	Identity   IdentityConfig   `json:"identity"`
	Range      RangeConfig      `json:"range"`
	Volume     VolumeConfig     `json:"volume"`
	Time       TimeConfig       `json:"time"`
	Content    ContentConfig    `json:"content"`
}

type RepositoryConfig struct {
	Path       string `json:"path"`
	Init       bool   `json:"init"`
	Branch     string `json:"branch"`
	Remote     string `json:"remote"`
	Push       bool   `json:"push"`
	AllowDirty bool   `json:"allow_dirty"`
}

type IdentityConfig struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type RangeConfig struct {
	Start        time.Time `json:"-"`
	End          time.Time `json:"-"`
	StartRaw     string    `json:"start"`
	EndRaw       string    `json:"end"`
	Timezone     string    `json:"timezone"`
	SkipWeekends bool      `json:"skip_weekends"`
}

type VolumeConfig struct {
	MinCommitsPerDay  int     `json:"min_commits_per_day"`
	MaxCommitsPerDay  int     `json:"max_commits_per_day"`
	ActiveDayProb     float64 `json:"active_day_probability"`
	WeekendMultiplier float64 `json:"weekend_multiplier"`
}

type TimeConfig struct {
	Start    time.Duration `json:"-"`
	End      time.Duration `json:"-"`
	StartRaw string        `json:"start"`
	EndRaw   string        `json:"end"`
}

type ContentConfig struct {
	ActivityFile     string   `json:"activity_file"`
	LineTemplate     string   `json:"line_template"`
	MessageTemplates []string `json:"message_templates"`
}

func Load(path string) (Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := defaultConfig()
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if strings.TrimSpace(cfg.Repository.Path) != "" && !filepath.IsAbs(cfg.Repository.Path) {
		cfg.Repository.Path = filepath.Join(filepath.Dir(path), cfg.Repository.Path)
	}

	if err := hydrate(&cfg); err != nil {
		return Config{}, err
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Repository: RepositoryConfig{
			Branch: "main",
		},
		Range: RangeConfig{
			Timezone: "+00:00",
		},
		Volume: VolumeConfig{
			MinCommitsPerDay:  1,
			MaxCommitsPerDay:  1,
			ActiveDayProb:     1,
			WeekendMultiplier: 1,
		},
		Time: TimeConfig{
			StartRaw: "09:00",
			EndRaw:   "17:00",
		},
		Content: ContentConfig{
			ActivityFile:     "activity.log",
			LineTemplate:     "{date} {time} synthetic event #{sequence}",
			MessageTemplates: []string{"Synthetic commit {date} #{index}"},
		},
	}
}

func hydrate(cfg *Config) error {
	start, err := time.Parse("2006-01-02", cfg.Range.StartRaw)
	if err != nil {
		return fmt.Errorf("range.start must be an ISO date like 2010-01-31")
	}
	end, err := time.Parse("2006-01-02", cfg.Range.EndRaw)
	if err != nil {
		return fmt.Errorf("range.end must be an ISO date like 2010-01-31")
	}
	startTime, err := parseClock(cfg.Time.StartRaw)
	if err != nil {
		return fmt.Errorf("time.start must be a time like 09:30")
	}
	endTime, err := parseClock(cfg.Time.EndRaw)
	if err != nil {
		return fmt.Errorf("time.end must be a time like 17:30")
	}

	cfg.Range.Start = start
	cfg.Range.End = end
	cfg.Time.Start = startTime
	cfg.Time.End = endTime
	return nil
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.Repository.Path) == "" {
		return fmt.Errorf("repository.path is required")
	}
	if strings.TrimSpace(cfg.Repository.Branch) == "" {
		return fmt.Errorf("repository.branch is required")
	}
	if strings.TrimSpace(cfg.Identity.Name) == "" {
		return fmt.Errorf("identity.name is required")
	}
	if strings.TrimSpace(cfg.Identity.Email) == "" {
		return fmt.Errorf("identity.email is required")
	}
	if cfg.Range.End.Before(cfg.Range.Start) {
		return fmt.Errorf("range.end must be on or after range.start")
	}
	if !regexp.MustCompile(`^[+-]\d{2}:\d{2}$`).MatchString(cfg.Range.Timezone) {
		return fmt.Errorf("range.timezone must use +HH:MM or -HH:MM format")
	}
	if cfg.Volume.MinCommitsPerDay < 0 {
		return fmt.Errorf("volume.min_commits_per_day must be >= 0")
	}
	if cfg.Volume.MaxCommitsPerDay < cfg.Volume.MinCommitsPerDay {
		return fmt.Errorf("volume.max_commits_per_day must be >= min_commits_per_day")
	}
	if cfg.Volume.ActiveDayProb < 0 || cfg.Volume.ActiveDayProb > 1 {
		return fmt.Errorf("volume.active_day_probability must be between 0 and 1")
	}
	if cfg.Volume.WeekendMultiplier < 0 {
		return fmt.Errorf("volume.weekend_multiplier must be >= 0")
	}
	if cfg.Time.End <= cfg.Time.Start {
		return fmt.Errorf("time.end must be after time.start")
	}
	if len(cfg.Content.MessageTemplates) == 0 {
		return fmt.Errorf("content.message_templates must contain at least one template")
	}
	if !isSafeRelativePath(cfg.Content.ActivityFile) {
		return fmt.Errorf("content.activity_file must be a relative path inside the target repository")
	}
	return nil
}

func parseClock(value string) (time.Duration, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return time.Duration(parsed.Hour())*time.Hour + time.Duration(parsed.Minute())*time.Minute, nil
}

func isSafeRelativePath(value string) bool {
	clean := filepath.Clean(value)
	if strings.TrimSpace(value) == "" || clean == "." || filepath.IsAbs(value) {
		return false
	}
	for _, part := range strings.Split(clean, string(os.PathSeparator)) {
		if part == ".." {
			return false
		}
	}
	return true
}
