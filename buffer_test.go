package logs

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestBufferCircular(t *testing.T) {
	m := New(WithMaxEntries(1000))

	for i := 1; i <= 1001; i++ {
		m.Append(Entry{Message: fmt.Sprintf("%d", i)})
	}

	if got := m.Len(); got != 1000 {
		t.Fatalf("Len() = %d, quer 1000", got)
	}

	entries := m.Entries()
	if len(entries) != 1000 {
		t.Fatalf("len(Entries()) = %d, quer 1000", len(entries))
	}
	if entries[0].Message != "2" {
		t.Errorf("primeira entrada = %q, quer %q", entries[0].Message, "2")
	}
	if entries[len(entries)-1].Message != "1001" {
		t.Errorf("última entrada = %q, quer %q", entries[len(entries)-1].Message, "1001")
	}
}

func TestBufferDefaultCapacity(t *testing.T) {
	m := New()
	for i := 0; i < defaultMaxEntries+5; i++ {
		m.Append(Entry{Message: fmt.Sprintf("%d", i)})
	}

	if got := m.Len(); got != defaultMaxEntries {
		t.Fatalf("Len() = %d, quer %d", got, defaultMaxEntries)
	}
}

func TestWithMaxEntriesClamp(t *testing.T) {
	m := New(WithMaxEntries(1))
	for i := 0; i < 20; i++ {
		m.Append(Entry{Message: fmt.Sprintf("%d", i)})
	}

	if got := m.Len(); got != minMaxEntries {
		t.Fatalf("Len() = %d, quer %d", got, minMaxEntries)
	}
	entries := m.Entries()
	if entries[0].Message != "10" {
		t.Errorf("primeira entrada = %q, quer %q", entries[0].Message, "10")
	}
}

func TestAppendEmptyReturnsNil(t *testing.T) {
	m := New()
	if cmd := m.Append(); cmd != nil {
		t.Fatal("Append() sem entradas deveria devolver nil")
	}
	if m.Len() != 0 {
		t.Fatalf("Len() = %d, quer 0", m.Len())
	}
}

func TestAppendReturnsCommand(t *testing.T) {
	m := New()
	cmd := m.Append(Entry{Message: "a"}, Entry{Message: "b"})
	if cmd == nil {
		t.Fatal("Append() deveria devolver um comando")
	}

	msg := cmd()
	appended, ok := msg.(AppendedMsg)
	if !ok {
		t.Fatalf("mensagem = %T, quer AppendedMsg", msg)
	}
	if appended.Count != 2 {
		t.Errorf("Count = %d, quer 2", appended.Count)
	}
}

func TestAppendf(t *testing.T) {
	m := New()
	m.Appendf(Warn, "tentativa %d de %d", 2, 3)

	entries := m.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(Entries()) = %d, quer 1", len(entries))
	}
	if entries[0].Level != Warn {
		t.Errorf("Level = %v, quer Warn", entries[0].Level)
	}
	if entries[0].Message != "tentativa 2 de 3" {
		t.Errorf("Message = %q", entries[0].Message)
	}
	if entries[0].Time.IsZero() {
		t.Error("Appendf deveria preencher Time")
	}
}

func TestEntriesReturnsCopy(t *testing.T) {
	m := New()
	m.Append(Entry{Message: "original"})

	entries := m.Entries()
	entries[0].Message = "alterada"

	if got := m.Entries()[0].Message; got != "original" {
		t.Errorf("Message = %q, quer %q", got, "original")
	}
}

func TestClear(t *testing.T) {
	m := New(WithMaxEntries(50))
	m.SetMinLevel(Warn)
	m.SetFilter("x")
	m.Append(Entry{Message: "x"}, Entry{Message: "outra"})

	m.Clear()

	if got := m.Len(); got != 0 {
		t.Errorf("Len() = %d, quer 0", got)
	}
	if len(m.Entries()) != 0 {
		t.Errorf("Entries() deveria estar vazio")
	}
	if m.Follow() != true {
		t.Error("Clear não deveria desligar o follow")
	}
	if m.Filter() != "x" {
		t.Errorf("Filter() = %q, quer %q", m.Filter(), "x")
	}
}

func TestZeroValueModel(t *testing.T) {
	var m Model

	if m.Len() != 0 {
		t.Errorf("Len() = %d, quer 0", m.Len())
	}
	if len(m.Entries()) != 0 {
		t.Errorf("Entries() deveria estar vazio")
	}

	m.Append(Entry{Message: "ok"})
	if m.Len() != 1 {
		t.Errorf("Len() = %d, quer 1", m.Len())
	}

	if got := m.View().Content; got != "" {
		t.Errorf("View() = %q, quer vazio sem tamanho", got)
	}

	m.SetSize(20, 4)
	if got := m.View().Content; got == "" {
		t.Error("View() não deveria ser vazia após SetSize")
	}
}

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := New()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 33, Height: 7})
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update devolveu %T", next)
	}
	if got.width != 33 || got.height != 7 {
		t.Errorf("tamanho = %dx%d, quer 33x7", got.width, got.height)
	}
}
