package logs

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFilterTextAndSource(t *testing.T) {
	m := New()
	m.SetSize(80, 10)
	m.Append(
		Entry{Level: Info, Source: "Download", Message: "AÇÃO concluída"},
		Entry{Level: Info, Source: "api", Message: "outra coisa"},
		Entry{Level: Warn, Source: "Download", Message: "falha ao baixar"},
	)

	tests := []struct {
		filter string
		want   int
	}{
		{"", 3},
		{"ação", 1},
		{"AÇÃO", 1},
		{"Ação", 1},
		{"download", 2},
		{"DOWNLOAD", 2},
		{"api", 1},
		{"inexistente", 0},
	}

	for _, tt := range tests {
		m.SetFilter(tt.filter)
		if got := len(m.Entries()); got != tt.want {
			t.Errorf("filtro %q: %d entradas, quer %d", tt.filter, got, tt.want)
		}
	}
}

func TestFilterDoesNotDiscardBuffer(t *testing.T) {
	m := New()
	m.Append(Entry{Message: "alpha"}, Entry{Message: "beta"})
	m.SetFilter("alpha")

	if got := m.Len(); got != 2 {
		t.Errorf("Len() = %d, quer 2", got)
	}
	if got := len(m.Entries()); got != 1 {
		t.Errorf("len(Entries()) = %d, quer 1", got)
	}

	m.SetFilter("")
	if got := len(m.Entries()); got != 2 {
		t.Errorf("len(Entries()) = %d, quer 2", got)
	}
}

func TestMinLevelFiltersRenderingOnly(t *testing.T) {
	m := New()
	m.Append(
		Entry{Level: Debug, Message: "d"},
		Entry{Level: Info, Message: "i"},
		Entry{Level: Warn, Message: "w"},
		Entry{Level: Error, Message: "e"},
	)

	m.SetMinLevel(Warn)
	if got := m.Len(); got != 4 {
		t.Errorf("Len() = %d, quer 4", got)
	}
	entries := m.Entries()
	if len(entries) != 2 || entries[0].Message != "w" || entries[1].Message != "e" {
		t.Errorf("Entries() = %+v, quer apenas w e e", entries)
	}

	m.SetMinLevel(Level(99))
	if m.minLevel != Error {
		t.Errorf("minLevel = %v, quer Error após clamp", m.minLevel)
	}
	if got := len(m.Entries()); got != 1 {
		t.Errorf("len(Entries()) = %d, quer 1", got)
	}

	m.SetMinLevel(Level(-99))
	if m.minLevel != Debug {
		t.Errorf("minLevel = %v, quer Debug após clamp", m.minLevel)
	}
	if got := len(m.Entries()); got != 4 {
		t.Errorf("len(Entries()) = %d, quer 4", got)
	}
}

func TestNavigationAndFollow(t *testing.T) {
	m := New()
	m.SetSize(60, 8) // status + separador + 6 entradas
	for i := 1; i <= 20; i++ {
		m.Append(Entry{Level: Info, Message: fmt.Sprintf("msg-%02d", i)})
	}

	if !m.Follow() {
		t.Fatal("New deveria começar com follow ligado")
	}

	content := m.View().Content
	if !strings.Contains(content, "msg-20") {
		t.Error("view inicial deveria mostrar a última entrada")
	}
	if strings.Contains(content, "msg-14") {
		t.Error("view inicial não deveria mostrar entradas fora da área")
	}

	m = testPress(t, m, "k")
	if m.Follow() {
		t.Error("k deveria desligar o follow")
	}
	if got := m.selectedIndex(m.visible()); got != 18 {
		t.Errorf("índice selecionado = %d, quer 18", got)
	}
	if line := selectedLine(m.View().Content); !strings.Contains(line, "msg-19") {
		t.Errorf("linha selecionada = %q, quer msg-19", line)
	}

	m = testPress(t, m, "j")
	if got := m.selectedIndex(m.visible()); got != 19 {
		t.Errorf("índice selecionado = %d, quer 19", got)
	}

	m = testPress(t, m, "g")
	if m.Follow() {
		t.Error("g não deveria reativar o follow")
	}
	if got := m.selectedIndex(m.visible()); got != 0 {
		t.Errorf("índice selecionado = %d, quer 0", got)
	}
	content = m.View().Content
	if !strings.Contains(content, "msg-01") {
		t.Error("view no topo deveria mostrar msg-01")
	}
	if strings.Contains(content, "msg-20") {
		t.Error("view no topo não deveria mostrar msg-20")
	}

	m = testPress(t, m, "G")
	if !m.Follow() {
		t.Error("G deveria reativar o follow")
	}
	if got := m.selectedIndex(m.visible()); got != 19 {
		t.Errorf("índice selecionado = %d, quer 19", got)
	}

	m = testPress(t, m, "k")
	m = testPress(t, m, "up")
	if got := m.selectedIndex(m.visible()); got != 17 {
		t.Errorf("índice selecionado = %d, quer 17", got)
	}
}

func TestFollowIsStableWhenEntriesArrive(t *testing.T) {
	m := New()
	m.SetSize(60, 8)
	for i := 1; i <= 20; i++ {
		m.Append(Entry{Message: fmt.Sprintf("msg-%02d", i)})
	}

	m = testPress(t, m, "k")
	before := selectedLine(m.View().Content)

	m.Append(Entry{Message: "msg-21"})
	m.Append(Entry{Message: "msg-22"})

	if got := selectedLine(m.View().Content); got != before {
		t.Errorf("seleção mudou de %q para %q após novas entradas", before, got)
	}
	if !strings.Contains(before, "msg-19") {
		t.Errorf("seleção = %q, quer msg-19", before)
	}
}

func TestFollowToggleKey(t *testing.T) {
	m := New()
	m.SetSize(40, 6)
	for i := 1; i <= 5; i++ {
		m.Append(Entry{Message: fmt.Sprintf("m%d", i)})
	}

	m = testPress(t, m, "g")
	if m.Follow() {
		t.Fatal("g deveria desligar o follow")
	}

	m = testPress(t, m, "f")
	if !m.Follow() {
		t.Fatal("f deveria ligar o follow")
	}
	if got := m.selectedIndex(m.visible()); got != 4 {
		t.Errorf("índice selecionado = %d, quer 4", got)
	}

	m = testPress(t, m, "f")
	if m.Follow() {
		t.Fatal("f deveria desligar o follow")
	}
}

func TestSearchMode(t *testing.T) {
	m := New()
	m.SetSize(60, 8)
	m.Append(
		Entry{Message: "alpha"},
		Entry{Message: "BETA"},
		Entry{Message: "Ação"},
	)

	m = testPress(t, m, "/")
	if !m.searching {
		t.Fatal("/ deveria entrar no modo de busca")
	}

	m = testPress(t, m, "b")
	m = testPress(t, m, "e")
	if got := m.Filter(); got != "be" {
		t.Fatalf("Filter() = %q, quer %q", got, "be")
	}
	if !strings.Contains(m.View().Content, "filtro: be") {
		t.Errorf("status deveria mostrar o filtro em edição: %q", m.View().Content)
	}

	m = testPress(t, m, "enter")
	if m.searching {
		t.Fatal("enter deveria confirmar a busca")
	}
	if got := m.Filter(); got != "be" {
		t.Errorf("Filter() = %q, quer %q", got, "be")
	}
	entries := m.Entries()
	if len(entries) != 1 || entries[0].Message != "BETA" {
		t.Errorf("Entries() = %+v, quer apenas BETA", entries)
	}
}

func TestSearchCancelRestoresFilter(t *testing.T) {
	m := New()
	m.SetFilter("alpha")

	m = testPress(t, m, "/")
	if got := m.Filter(); got != "" {
		t.Errorf("Filter() durante a busca = %q, quer vazio", got)
	}

	m = testPress(t, m, "x")
	m = testPress(t, m, "esc")
	if m.searching {
		t.Fatal("esc deveria cancelar a busca")
	}
	if got := m.Filter(); got != "alpha" {
		t.Errorf("Filter() = %q, quer %q", got, "alpha")
	}
}

func TestSearchBackspaceHandlesMultibyte(t *testing.T) {
	m := New()
	m = testPress(t, m, "/")
	m = testPress(t, m, "a")
	m = testPress(t, m, "ç")

	if got := m.Filter(); got != "aç" {
		t.Fatalf("Filter() = %q, quer %q", got, "aç")
	}

	m = testPress(t, m, "backspace")
	if got := m.Filter(); got != "a" {
		t.Errorf("Filter() = %q, quer %q", got, "a")
	}
}

func TestSearchTypingDoesNotNavigate(t *testing.T) {
	m := New()
	m.SetSize(40, 6)
	for i := 1; i <= 10; i++ {
		m.Append(Entry{Message: fmt.Sprintf("m%d", i)})
	}

	m = testPress(t, m, "/")
	for _, ch := range []string{"g", "j", "k", "G", "l", "c"} {
		m = testPress(t, m, ch)
	}
	if got := m.Filter(); got != "gjkGlc" {
		t.Errorf("Filter() = %q, quer %q", got, "gjkGlc")
	}
	if m.Len() != 10 {
		t.Errorf("Len() = %d, quer 10; teclas de busca não devem limpar o buffer", m.Len())
	}
	if !m.searching {
		t.Error("a busca deveria continuar ativa")
	}
}

func TestCycleLevelKey(t *testing.T) {
	m := New()

	want := []Level{Info, Warn, Error, Debug}
	for i, expected := range want {
		m = testPress(t, m, "l")
		if m.minLevel != expected {
			t.Errorf("ciclo %d: minLevel = %v, quer %v", i, m.minLevel, expected)
		}
	}
}

func TestClearKey(t *testing.T) {
	m := New()
	m.Append(Entry{Message: "a"}, Entry{Message: "b"})

	m = testPress(t, m, "c")
	if m.Len() != 0 {
		t.Errorf("Len() = %d, quer 0", m.Len())
	}
	if got := m.View().Content; !strings.Contains(got, "(sem entradas)") {
		t.Errorf("View() = %q, deveria mostrar o placeholder", got)
	}
}

func TestSetFilterSanitizesControlCharacters(t *testing.T) {
	m := New()
	m.SetFilter("a\nb\tc\x00d")

	if got := m.Filter(); got != "a b c d" {
		t.Errorf("Filter() = %q, quer %q", got, "a b c d")
	}
	if strings.Count(m.View().Content, "\n") != 2 {
		t.Errorf("a linha de status não deveria quebrar: %q", m.View().Content)
	}
}

func TestSetSizeNegative(t *testing.T) {
	m := New()
	m.SetSize(-10, -20)

	if m.width != 0 || m.height != 0 {
		t.Errorf("tamanho = %dx%d, quer 0x0", m.width, m.height)
	}
	if got := m.View().Content; got != "" {
		t.Errorf("View() = %q, quer vazio", got)
	}
}

func TestWindowSizeIsHandled(t *testing.T) {
	m := New()
	m.Append(Entry{Message: "ok"})
	next, _ := testUpdate(t, m, tea.WindowSizeMsg{Width: 21, Height: 4})
	if next.width != 21 || next.height != 4 {
		t.Errorf("tamanho = %dx%d, quer 21x4", next.width, next.height)
	}
	assertLineWidths(t, next.View().Content, 21)
}
