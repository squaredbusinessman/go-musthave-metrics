package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// DurationSeconds хранит продолжительность в целых секундах.
type DurationSeconds int

// Seconds возвращает длительность в секундах.
func (d DurationSeconds) Seconds() int {
	return int(d)
}

// UnmarshalJSON поддерживает либо строку с time.ParseDuration, либо число секунд.
func (d *DurationSeconds) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		duration, err := time.ParseDuration(asString)
		if err != nil {
			return fmt.Errorf("parse duration %q: %w", asString, err)
		}
		if duration%time.Second != 0 {
			return fmt.Errorf("duration %q must be a whole number of seconds", asString)
		}
		*d = DurationSeconds(duration / time.Second)
		return nil
	}

	var asInt int
	if err := json.Unmarshal(data, &asInt); err == nil {
		*d = DurationSeconds(asInt)
		return nil
	}

	return fmt.Errorf("duration must be a string or integer")
}

// DiscoverPath находит путь к конфигурации по флагам -c/-config или переменной окружения.
func DiscoverPath(args []string, envVar string) (string, error) {
	path, ok, err := discoverPathFromArgs(args)
	if err != nil {
		return "", err
	}
	if ok {
		return path, nil
	}
	return os.Getenv(envVar), nil
}

// ReadJSONFile загружает JSON-конфигурацию и запрещает неизвестные поля.
func ReadJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode config file %q: %w", path, err)
	}

	return nil
}

func discoverPathFromArgs(args []string) (string, bool, error) {
	var (
		path  string
		found bool
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-c" || arg == "-config":
			if i+1 >= len(args) {
				return "", false, fmt.Errorf("flag %s requires a value", arg)
			}
			path = args[i+1]
			found = true
			i++
		case strings.HasPrefix(arg, "-c="):
			path = strings.TrimPrefix(arg, "-c=")
			found = true
		case strings.HasPrefix(arg, "-config="):
			path = strings.TrimPrefix(arg, "-config=")
			found = true
		}
	}

	return path, found, nil
}
