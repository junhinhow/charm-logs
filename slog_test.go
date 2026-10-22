package logs

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSlogHandlerLevelsAndAttributes(t *testing.T) {
	m := New()
	logger := slog.New(m.SlogHandler())

	logger.Debug("detalhe")
	logger.Info("olá", "user", "ana")
	logger.Warn("cuidado", "tentativa", 2)
	logger.Error("falhou", "err", errors.New("boom"))

	entries := m.Entries()
	if len(entries) != 4 {
		t.Fatalf("len(Entries()) = %d, quer 4", len(entries))
	}

	wantLevels := []Level{Debug, Info, Warn, Error}
	wantMessages := []string{
		"detalhe",
		"olá user=ana",
		"cuidado tentativa=2",
		"falhou err=boom",
	}
	for i := range entries {
		if entries[i].Level != wantLevels[i] {
			t.Errorf("entrada %d: Level = %v, quer %v", i, entries[i].Level, wantLevels[i])
		}
		if entries[i].Message != wantMessages[i] {
			t.Errorf("entrada %d: Message = %q, quer %q", i, entries[i].Message, wantMessages[i])
		}
		if entries[i].Time.IsZero() {
			t.Errorf("entrada %d: Time não deveria ser zero", i)
		}
	}
}

func TestSlogHandlerWithAttrsAndGroups(t *testing.T) {
	m := New()
	logger := slog.New(m.SlogHandler()).
		With("app", "demo").
		WithGroup("req")

	logger.Info("ok", "id", 7, slog.Group("user", slog.String("name", "ana")))

	entries := m.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(Entries()) = %d, quer 1", len(entries))
	}
	want := "ok app=demo req.id=7 req.user.name=ana"
	if got := entries[0].Message; got != want {
		t.Errorf("Message = %q, quer %q", got, want)
	}
}

func TestSlogHandlerWithAttrsIsImmutable(t *testing.T) {
	m := New()
	base := m.SlogHandler()
	derived := base.WithAttrs([]slog.Attr{slog.String("app", "demo")})

	slog.New(base).Info("base")
	slog.New(derived).Info("derivado")

	entries := m.Entries()
	if len(entries) != 2 {
		t.Fatalf("len(Entries()) = %d, quer 2", len(entries))
	}
	if entries[0].Message != "base" {
		t.Errorf("base = %q, não deveria receber atributos", entries[0].Message)
	}
	if entries[1].Message != "derivado app=demo" {
		t.Errorf("derivado = %q", entries[1].Message)
	}
}

func TestSlogHandlerCustomLevels(t *testing.T) {
	m := New()
	logger := slog.New(m.SlogHandler())
	ctx := context.Background()

	logger.Log(ctx, slog.Level(-8), "muito debug")
	logger.Log(ctx, slog.Level(2), "quase info")
	logger.Log(ctx, slog.Level(6), "quase warn")
	logger.Log(ctx, slog.Level(12), "acima de error")

	want := []Level{Debug, Info, Warn, Error}
	entries := m.Entries()
	for i := range want {
		if entries[i].Level != want[i] {
			t.Errorf("entrada %d: Level = %v, quer %v", i, entries[i].Level, want[i])
		}
	}
}

func TestSlogHandlerZeroTime(t *testing.T) {
	m := New()
	h := m.SlogHandler()

	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "sem tempo", 0)
	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle devolveu erro: %v", err)
	}

	entries := m.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(Entries()) = %d, quer 1", len(entries))
	}
	if entries[0].Time.IsZero() {
		t.Error("Handle deveria preencher Time quando o registro não traz horário")
	}
}

func TestSlogHandlerConcurrent(t *testing.T) {
	m := New(WithMaxEntries(1000))
	logger := slog.New(m.SlogHandler())

	const writers = 20
	const perWriter = 100

	var wg sync.WaitGroup
	wg.Add(writers + 1)

	for w := 0; w < writers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				logger.Info("concorrente", "writer", id, "i", i)
			}
		}(w)
	}

	// Leitores concorrentes exercitam o mutex do buffer durante as escritas.
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = m.Entries()
			_ = m.View().Content
		}
	}()

	wg.Wait()

	if got := m.Len(); got != 1000 {
		t.Errorf("Len() = %d, quer 1000", got)
	}
	for _, e := range m.Entries() {
		if !strings.Contains(e.Message, "concorrente") {
			t.Fatalf("mensagem inesperada: %q", e.Message)
		}
	}
}

func TestSlogHandlerNilModel(t *testing.T) {
	var m *Model
	h := m.SlogHandler()
	if h == nil {
		t.Fatal("SlogHandler não deveria devolver nil")
	}
	if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "x", 0)); err != nil {
		t.Errorf("Handle devolveu erro: %v", err)
	}
}

func TestSlogHandlerAppendsToModelCopies(t *testing.T) {
	m := New()
	copyOfModel := m

	logger := slog.New(copyOfModel.SlogHandler())
	logger.Warn("via cópia")

	if got := m.Len(); got != 1 {
		t.Fatalf("Len() = %d, quer 1; o buffer deveria ser compartilhado", got)
	}
	if got := m.Entries()[0].Level; got != Warn {
		t.Errorf("Level = %v, quer Warn", got)
	}
}
