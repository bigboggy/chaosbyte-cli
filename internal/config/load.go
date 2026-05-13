package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

// tomlRoomConfig mirrors RoomConfig with string colors so a TOML file can
// hold hex strings without lipgloss types. The Load functions convert from
// this representation into the runtime struct.
type tomlRoomConfig struct {
	Slug      string          `toml:"slug"`
	Brand     tomlBrand       `toml:"brand"`
	Theme     tomlTheme       `toml:"theme"`
	Mod       tomlMod         `toml:"mod"`
	Surfaces  tomlSurfaces    `toml:"surfaces"`
	Spotlight tomlSpotlight   `toml:"spotlight"`
}

type tomlBrand struct {
	Name    string `toml:"name"`
	MOTD    string `toml:"motd"`
	Tagline string `toml:"tagline"`
}

type tomlTheme struct {
	Bg       string `toml:"bg"`
	Fg       string `toml:"fg"`
	Muted    string `toml:"muted"`
	Accent   string `toml:"accent"`
	Accent2  string `toml:"accent2"`
	BorderHi string `toml:"border_hi"`
	BorderLo string `toml:"border_lo"`
}

type tomlMod struct {
	Nick               string `toml:"nick"`
	Welcome            string `toml:"welcome"`
	PromptIntervalSecs int    `toml:"prompt_interval_secs"`
	IdleThresholdSecs  int    `toml:"idle_threshold_secs"`
}

type tomlSurfaces struct {
	Chat        bool `toml:"chat"`
	Spotlight   bool `toml:"spotlight"`
	Games       bool `toml:"games"`
	Skills      bool `toml:"skills"`
	Discussions bool `toml:"discussions"`
}

type tomlSpotlight struct {
	Name        string `toml:"name"`
	Author      string `toml:"author"`
	Description string `toml:"description"`
	RepoURL     string `toml:"repo_url"`
}

// LoadFromFile reads a single .toml file and returns the team's
// RoomConfig with the flagship defaults filled in for missing fields. The
// file's parent directory becomes the team's slug if no slug field is
// set, so a directory layout of configs/<slug>/room.toml is supported.
func LoadFromFile(path string) (RoomConfig, error) {
	var raw tomlRoomConfig
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return RoomConfig{}, fmt.Errorf("config: %w", err)
	}
	cfg := tomlToRoom(raw)
	if cfg.Slug == "" {
		// Derive slug from filename: "configs/acme.toml" → "acme".
		base := filepath.Base(path)
		cfg.Slug = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return MergeWithDefaults(cfg), nil
}

// LoadFromDir reads every .toml file in the directory as a team config.
// Returns the loaded configs in directory order. Errors on any single
// file are returned with the slug for diagnostics; loading continues
// past the bad file so one team's malformed config doesn't block others.
func LoadFromDir(dir string) ([]RoomConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("config: read dir %q: %w", dir, err)
	}
	var out []RoomConfig
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		cfg, err := LoadFromFile(path)
		if err != nil {
			return out, fmt.Errorf("config %q: %w", path, err)
		}
		out = append(out, cfg)
	}
	return out, nil
}

func tomlToRoom(raw tomlRoomConfig) RoomConfig {
	return RoomConfig{
		Slug: raw.Slug,
		Brand: BrandConfig{
			Name:    raw.Brand.Name,
			MOTD:    raw.Brand.MOTD,
			Tagline: raw.Brand.Tagline,
		},
		Theme: ThemeConfig{
			Bg:       parseColor(raw.Theme.Bg),
			Fg:       parseColor(raw.Theme.Fg),
			Muted:    parseColor(raw.Theme.Muted),
			Accent:   parseColor(raw.Theme.Accent),
			Accent2:  parseColor(raw.Theme.Accent2),
			BorderHi: parseColor(raw.Theme.BorderHi),
			BorderLo: parseColor(raw.Theme.BorderLo),
		},
		Mod: ModConfig{
			Nick:               raw.Mod.Nick,
			Welcome:            raw.Mod.Welcome,
			PromptIntervalSecs: raw.Mod.PromptIntervalSecs,
			IdleThresholdSecs:  raw.Mod.IdleThresholdSecs,
		},
		Surfaces: SurfacesConfig{
			Chat:        raw.Surfaces.Chat,
			Spotlight:   raw.Surfaces.Spotlight,
			Games:       raw.Surfaces.Games,
			Skills:      raw.Surfaces.Skills,
			Discussions: raw.Surfaces.Discussions,
		},
		Spotlight: SpotlightConfig{
			Name:        raw.Spotlight.Name,
			Author:      raw.Spotlight.Author,
			Description: raw.Spotlight.Description,
			RepoURL:     raw.Spotlight.RepoURL,
		},
	}
}

func parseColor(s string) lipgloss.Color {
	if s == "" {
		return lipgloss.Color("")
	}
	return lipgloss.Color(s)
}
