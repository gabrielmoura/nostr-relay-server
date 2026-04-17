package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

type OutputFormat string

const (
	FormatYAML OutputFormat = "yaml"
	FormatJSON OutputFormat = "json"
)

func ParseOutputFormat(raw string) (OutputFormat, error) {
	format := OutputFormat(strings.ToLower(strings.TrimSpace(raw)))
	switch format {
	case FormatYAML, FormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("invalid --format %q: expected yaml or json", raw)
	}
}

func PrintDefaults(format OutputFormat) error {
	cfg, err := config.DefaultConfig()
	if err != nil {
		return fmt.Errorf("build default config: %w", err)
	}

	return printStruct(cfg, format)
}

func PrintEffective(filePath string, format OutputFormat) error {
	if strings.TrimSpace(filePath) == "" {
		if err := config.LoadConfig(); err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	} else {
		if err := config.LoadConfigFromFile(filePath); err != nil {
			return fmt.Errorf("load config file %q: %w", filePath, err)
		}
	}

	return printStruct(config.Cfg, format)
}

func WriteDefaults(filePath string, force bool) error {
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("--file cannot be empty")
	}

	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("file %q already exists (use --force to overwrite)", filePath)
		}
	}

	if err := config.WriteYamlConfig(filePath); err != nil {
		return fmt.Errorf("write config file %q: %w", filePath, err)
	}

	return nil
}

func ValidateConfig(filePath string) error {
	if strings.TrimSpace(filePath) == "" {
		if err := config.LoadConfig(); err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	} else {
		if err := config.LoadConfigFromFile(filePath); err != nil {
			return fmt.Errorf("load config file %q: %w", filePath, err)
		}
	}

	if config.Cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	if err := validateCronSchedules(config.Cfg); err != nil {
		return err
	}

	if errs := config.Cfg.RelayInformation.Check(); len(errs) > 0 {
		return fmt.Errorf("relay_information validation failed: %v", errs)
	}

	return nil
}

func validateCronSchedules(cfg *config.Config) error {
	parser := cron.NewParser(
		cron.Second |
			cron.Minute |
			cron.Hour |
			cron.Dom |
			cron.Month |
			cron.Dow,
	)

	checks := []struct {
		name     string
		enabled  bool
		schedule string
	}{
		{name: "cron.db_optimization", enabled: cfg.Cron.DBOptimization.Enabled, schedule: cfg.Cron.DBOptimization.Schedule},
		{name: "cron.reported_events_fetch", enabled: cfg.Cron.ReportedEventsFetch.Enabled, schedule: cfg.Cron.ReportedEventsFetch.Schedule},
		{name: "cron.delete_old_events", enabled: cfg.Cron.DeleteOldEvents.Enabled, schedule: cfg.Cron.DeleteOldEvents.Schedule},
		{name: "cron.nip40", enabled: cfg.Cron.NIP40.Enabled, schedule: cfg.Cron.NIP40.Schedule},
	}

	for _, check := range checks {
		if !check.enabled {
			continue
		}
		if strings.TrimSpace(check.schedule) == "" {
			return fmt.Errorf("%s is enabled but schedule is empty", check.name)
		}
		if _, err := parser.Parse(check.schedule); err != nil {
			return fmt.Errorf("invalid schedule for %s: %w", check.name, err)
		}
	}

	if cfg.Cron.ReportedEventsFetch.Enabled && len(cfg.Cron.ReportedEventsFetch.Relays) == 0 {
		return fmt.Errorf("cron.reported_events_fetch is enabled but relays is empty")
	}

	return nil
}

func printStruct(v any, format OutputFormat) error {
	data, err := marshalStruct(v, format)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func marshalStruct(v any, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatYAML:
		data, err := yaml.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal yaml: %w", err)
		}
		return data, nil
	case FormatJSON:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal json: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}
