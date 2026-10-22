package logs

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// testKey converte um rótulo legível em tea.KeyPressMsg.
func testKey(label string) tea.KeyPressMsg {
	switch label {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	default:
		runes := []rune(label)
		return tea.KeyPressMsg{Code: runes[0], Text: label}
	}
}

// testUpdate aplica uma mensagem e devolve o Model resultante.
func testUpdate(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update devolveu %T, quer Model", next)
	}
	return got, cmd
}

// testPress envia uma tecla e devolve apenas o Model.
func testPress(t *testing.T, m Model, label string) Model {
	t.Helper()
	got, _ := testUpdate(t, m, testKey(label))
	return got
}

// selectedLine devolve a linha marcada como selecionada, sem ANSI.
func selectedLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "▸") {
			return ansi.Strip(line)
		}
	}
	return ""
}

// assertLineWidths falha se alguma linha exceder a largura informada.
func assertLineWidths(t *testing.T, content string, width int) {
	t.Helper()
	if content == "" {
		return
	}
	for i, line := range strings.Split(content, "\n") {
		if got := ansi.StringWidth(line); got > width {
			t.Errorf("linha %d tem largura %d, máximo %d: %q", i, got, width, line)
		}
	}
}

// assertValidUTF8 falha se o conteúdo não for UTF-8 válido.
func assertValidUTF8(t *testing.T, content string) {
	t.Helper()
	if !utf8.ValidString(content) {
		t.Fatal("conteúdo não é UTF-8 válido")
	}
}
