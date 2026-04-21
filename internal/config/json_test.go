package config

import (
	"encoding/json"
	"testing"
)

func TestDiscoverPath(t *testing.T) {
	t.Setenv("CONFIG", "/tmp/from-env.json")

	path, err := DiscoverPath(nil, "CONFIG")
	if err != nil {
		t.Fatalf("DiscoverPath() error = %v", err)
	}
	if got, want := path, "/tmp/from-env.json"; got != want {
		t.Fatalf("DiscoverPath() = %q, want %q", got, want)
	}

	path, err = DiscoverPath([]string{"-c", "/tmp/from-flag.json"}, "CONFIG")
	if err != nil {
		t.Fatalf("DiscoverPath() error = %v", err)
	}
	if got, want := path, "/tmp/from-flag.json"; got != want {
		t.Fatalf("DiscoverPath() = %q, want %q", got, want)
	}
}

func TestDurationSecondsUnmarshalJSON(t *testing.T) {
	var fromString DurationSeconds
	if err := json.Unmarshal([]byte(`"15s"`), &fromString); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got, want := fromString.Seconds(), 15; got != want {
		t.Fatalf("DurationSeconds = %d, want %d", got, want)
	}

	var fromInt DurationSeconds
	if err := json.Unmarshal([]byte(`7`), &fromInt); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got, want := fromInt.Seconds(), 7; got != want {
		t.Fatalf("DurationSeconds = %d, want %d", got, want)
	}
}
