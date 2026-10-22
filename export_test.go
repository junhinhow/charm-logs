package logs

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExportFormat(t *testing.T) {
	m := New()
	m.Append(
		Entry{
			Time:    time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC),
			Level:   Info,
			Source:  "api",
			Message: "hello",
		},
		Entry{
			Time:    time.Date(2026, 10, 22, 12, 0, 1, 0, time.FixedZone("BRT", -3*60*60)),
			Level:   Error,
			Message: "falhou: ação",
		},
	)

	var b strings.Builder
	if err := m.Export(&b); err != nil {
		t.Fatalf("Export devolveu erro: %v", err)
	}

	want := "2026-10-22T12:00:00Z INFO [api] hello\n" +
		"2026-10-22T12:00:01-03:00 ERROR falhou: ação\n"
	if got := b.String(); got != want {
		t.Errorf("Export() =\n%q\nquer\n%q", got, want)
	}
}

func TestExportDeterministic(t *testing.T) {
	m := New()
	for i := 0; i < 50; i++ {
		m.Append(Entry{
			Time:    time.Date(2026, 10, 22, 12, 0, i%60, 0, time.UTC),
			Level:   Level(i % 4),
			Source:  "worker",
			Message: "linha",
		})
	}

	var first, second strings.Builder
	if err := m.Export(&first); err != nil {
		t.Fatal(err)
	}
	if err := m.Export(&second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Error("duas exportações seguidas produziram textos diferentes")
	}
}

func TestExportHasNoANSI(t *testing.T) {
	m := New()
	m.Append(Entry{
		Time:    time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC),
		Level:   Error,
		Message: "\x1b[31mvermelho\x1b[0m",
	})

	var b strings.Builder
	if err := m.Export(&b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "\x1b") {
		t.Errorf("Export() contém ANSI: %q", b.String())
	}
	if !strings.Contains(b.String(), "vermelho") {
		t.Errorf("Export() = %q, deveria conter o texto", b.String())
	}
}

func TestExportRespectsVisibleEntries(t *testing.T) {
	m := New()
	m.Append(
		Entry{Level: Debug, Message: "debug"},
		Entry{Level: Warn, Message: "falha um"},
		Entry{Level: Warn, Message: "falha dois", Source: "api"},
		Entry{Level: Error, Message: "outra"},
	)
	m.SetMinLevel(Warn)
	m.SetFilter("falha")

	var b strings.Builder
	if err := m.Export(&b); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("Export() produziu %d linhas, quer 2:\n%s", len(lines), b.String())
	}
	if !strings.Contains(lines[0], "WARN falha um") {
		t.Errorf("linha 0 = %q", lines[0])
	}
	if !strings.Contains(lines[1], "WARN [api] falha dois") {
		t.Errorf("linha 1 = %q", lines[1])
	}
}

func TestExportSanitizesNewlines(t *testing.T) {
	m := New()
	m.Append(Entry{
		Time:    time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC),
		Level:   Info,
		Message: "primeira\nsegunda",
	})

	var b strings.Builder
	if err := m.Export(&b); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(b.String(), "\n"); got != 1 {
		t.Errorf("Export() tem %d quebras de linha, quer 1: %q", got, b.String())
	}
	if !strings.Contains(b.String(), "primeira segunda") {
		t.Errorf("Export() = %q", b.String())
	}
}

func TestExportEmpty(t *testing.T) {
	m := New()
	var b strings.Builder
	if err := m.Export(&b); err != nil {
		t.Fatal(err)
	}
	if b.String() != "" {
		t.Errorf("Export() = %q, quer vazio", b.String())
	}
}

func TestExportNilWriter(t *testing.T) {
	m := New()
	if err := m.Export(nil); err == nil {
		t.Fatal("Export(nil) deveria devolver erro")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("falha simulada")
}

func TestExportPropagatesWriteError(t *testing.T) {
	m := New()
	m.Append(Entry{Message: "linha"})

	if err := m.Export(failingWriter{}); err == nil {
		t.Fatal("Export deveria propagar o erro de escrita")
	}
}

func TestExportKeyWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())

	m := New()
	m.Append(Entry{
		Time:    time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC),
		Level:   Info,
		Message: "exportado",
	})

	next, cmd := testUpdate(t, m, testKey("e"))
	if cmd == nil {
		t.Fatal("a tecla e deveria devolver um comando")
	}
	if next.Len() != 1 {
		t.Errorf("Len() = %d, quer 1", next.Len())
	}

	msg := cmd()
	exported, ok := msg.(ExportedMsg)
	if !ok {
		t.Fatalf("mensagem = %T, quer ExportedMsg", msg)
	}
	if exported.Err != nil {
		t.Fatalf("exportação falhou: %v", exported.Err)
	}
	if exported.Path != exportFileName {
		t.Errorf("Path = %q, quer %q", exported.Path, exportFileName)
	}

	data, err := os.ReadFile(exportFileName)
	if err != nil {
		t.Fatalf("arquivo não foi criado: %v", err)
	}
	want := "2026-10-22T12:00:00Z INFO exportado\n"
	if got := string(data); got != want {
		t.Errorf("arquivo = %q, quer %q", got, want)
	}
}
