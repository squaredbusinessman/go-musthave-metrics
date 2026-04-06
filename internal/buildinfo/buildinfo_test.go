package buildinfo

import (
	"bytes"
	"testing"
)

func TestPrint(t *testing.T) {
	t.Run("uses provided values", func(t *testing.T) {
		var buf bytes.Buffer

		Print(&buf, "1.2.3", "2026-04-06", "abc123")

		want := "Build version: 1.2.3\nBuild date: 2026-04-06\nBuild commit: abc123\n"
		if buf.String() != want {
			t.Fatalf("unexpected output:\nwant:\n%s\ngot:\n%s", want, buf.String())
		}
	})

	t.Run("uses fallback for empty values", func(t *testing.T) {
		var buf bytes.Buffer

		Print(&buf, "", "", "")

		want := "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n"
		if buf.String() != want {
			t.Fatalf("unexpected output:\nwant:\n%s\ngot:\n%s", want, buf.String())
		}
	})
}
