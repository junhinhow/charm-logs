package logs

import "testing"

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Debug, "debug"},
		{Info, "info"},
		{Warn, "warn"},
		{Error, "error"},
		{Level(99), "unknown"},
		{Level(-1), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, quer %q", tt.level, got, tt.want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", Debug},
		{"DEBUG", Debug},
		{" Info ", Info},
		{"warn", Warn},
		{"Warning", Warn},
		{"error", Error},
		{"ERRO", 0},
	}

	for _, tt := range tests {
		got, err := ParseLevel(tt.input)
		if tt.input == "ERRO" {
			if err == nil {
				t.Errorf("ParseLevel(%q) deveria falhar", tt.input)
			}
			if got != Info {
				t.Errorf("ParseLevel(%q) = %v, quer Info no erro", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLevel(%q) devolveu erro: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, quer %v", tt.input, got, tt.want)
		}
	}
}

func TestParseLevelEmpty(t *testing.T) {
	if _, err := ParseLevel(""); err == nil {
		t.Fatal("ParseLevel(\"\") deveria devolver erro")
	}
}

func TestClampLevel(t *testing.T) {
	if got := clampLevel(Level(-10)); got != Debug {
		t.Errorf("clampLevel(-10) = %v, quer Debug", got)
	}
	if got := clampLevel(Level(10)); got != Error {
		t.Errorf("clampLevel(10) = %v, quer Error", got)
	}
	if got := clampLevel(Warn); got != Warn {
		t.Errorf("clampLevel(Warn) = %v, quer Warn", got)
	}
}
