package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewFormats(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{format: FormatJSON, want: `"msg":"hello"`},
		{format: FormatText, want: `msg=hello`},
	}
	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			var buf bytes.Buffer
			log, err := New(&buf, "info", tc.format)
			if err != nil {
				t.Fatal(err)
			}
			log.Info("hello")
			if !strings.Contains(buf.String(), tc.want) {
				t.Fatalf("output %q does not contain %q", buf.String(), tc.want)
			}
		})
	}
}

func TestNewRejectsUnknownFormatAndLevel(t *testing.T) {
	if _, err := New(&bytes.Buffer{}, "info", "xml"); err == nil {
		t.Error("expected an error for an unknown format")
	}
	if _, err := New(&bytes.Buffer{}, "loud", FormatJSON); err == nil {
		t.Error("expected an error for an unknown level")
	}
}

func TestAtLeastDropsLowerLevels(t *testing.T) {
	var buf bytes.Buffer
	base, err := New(&buf, "debug", FormatText)
	if err != nil {
		t.Fatal(err)
	}
	quiet := AtLeast(base, slog.LevelInfo).With("component", "jobs")

	quiet.Debug("chatty")
	quiet.Info("important")
	base.Debug("app debug")

	out := buf.String()
	if strings.Contains(out, "chatty") {
		t.Error("debug record from the filtered logger should be dropped")
	}
	if !strings.Contains(out, "msg=important component=jobs") {
		t.Errorf("info record should pass with its attributes: %q", out)
	}
	if !strings.Contains(out, "app debug") {
		t.Error("the base logger must keep logging debug")
	}
}
