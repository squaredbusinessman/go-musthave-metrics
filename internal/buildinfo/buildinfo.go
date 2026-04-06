package buildinfo

import (
	"fmt"
	"io"
)

const fallbackValue = "N/A"

func valueOrFallback(value string) string {
	if value == "" {
		return fallbackValue
	}

	return value
}

func Print(w io.Writer, version, date, commit string) {
	fmt.Fprintf(
		w,
		"Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		valueOrFallback(version),
		valueOrFallback(date),
		valueOrFallback(commit),
	)
}
