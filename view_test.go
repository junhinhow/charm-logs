package logs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestViewInvalidSizes(t *testing.T) {
	sizes := [][2]int{
		{0, 10},
		{-1, 10},
		{10, 0},
		{10, -1},
		{0, 0},
		{-5, -5},
	}

	for _, size := range sizes {
		m := New()
		m.Append(Entry{Message: "não deveria aparecer"})
		m.SetSize(size[0], size[1])

		if got := m.View().Content; got != "" {
			t.Errorf("SetSize(%d, %d): View() = %q, quer vazio", size[0], size[1], got)
		}
	}
}

func TestViewNeverExceedsWidth(t *testing.T) {
	entries := []Entry{
		{Level: Info, Source: "download", Message: strings.Repeat("mensagem longa ", 10)},
		{Level: Error, Source: "api", Message: "falhou: " + strings.Repeat("🎉", 40)},
		{Level: Warn, Message: "sem origem"},
	}

	for _, width := range []int{20, 40} {
		m := New()
		m.SetSize(width, 10)
		m.Append(entries...)

		content := m.View().Content
		assertLineWidths(t, content, width)
		assertValidUTF8(t, content)
	}
}

func TestViewTinyWidths(t *testing.T) {
	for width := 1; width <= 25; width++ {
		m := New()
		m.SetSize(width, 6)
		m.Append(
			Entry{Level: Info, Source: "download", Message: "ação 🎉 concluída"},
			Entry{Level: Error, Source: "worker", Message: strings.Repeat("x", 50)},
		)

		content := m.View().Content
		assertLineWidths(t, content, width)
		assertValidUTF8(t, content)
	}
}

func TestViewHeight(t *testing.T) {
	for height := 1; height <= 10; height++ {
		m := New()
		m.SetSize(30, height)
		for i := 0; i < 20; i++ {
			m.Append(Entry{Message: fmt.Sprintf("linha %d", i)})
		}

		content := m.View().Content
		lines := 0
		if content != "" {
			lines = strings.Count(content, "\n") + 1
		}
		if lines > height {
			t.Errorf("altura %d: View() tem %d linhas", height, lines)
		}
		assertLineWidths(t, content, 30)
	}
}

func TestViewHeightOneShowsOnlyStatus(t *testing.T) {
	m := New()
	m.SetSize(40, 1)
	m.Append(Entry{Message: "conteúdo"})

	content := m.View().Content
	if strings.Contains(content, "\n") {
		t.Errorf("View() = %q, quer apenas o status", content)
	}
	if !strings.Contains(content, "1 entrada") {
		t.Errorf("View() = %q, deveria conter o status", content)
	}
	if strings.Contains(content, "conteúdo") {
		t.Errorf("View() = %q, não deveria conter entradas", content)
	}
}

func TestViewEmptyPlaceholder(t *testing.T) {
	m := New()
	m.SetSize(40, 5)

	content := m.View().Content
	if !strings.Contains(content, "(sem entradas)") {
		t.Errorf("View() = %q, deveria conter o placeholder", content)
	}
	if lines := strings.Count(content, "\n") + 1; lines != 3 {
		t.Errorf("View() tem %d linhas, quer 3", lines)
	}
}

func TestViewStatus(t *testing.T) {
	m := New()
	m.SetSize(80, 8)
	m.Append(
		Entry{Level: Debug, Message: "debug"},
		Entry{Level: Warn, Message: "warn x"},
		Entry{Level: Error, Message: "error"},
	)
	m.SetMinLevel(Warn)
	m.SetFilter("warn")
	m.SetFollow(false)

	status := strings.Split(m.View().Content, "\n")[0]
	for _, want := range []string{"3 entradas", "warn", "follow off", "filtro: warn"} {
		if !strings.Contains(status, want) {
			t.Errorf("status = %q, deveria conter %q", status, want)
		}
	}
}

func TestViewEmojiDoesNotBreakLayout(t *testing.T) {
	m := New()
	m.SetSize(20, 8)
	m.Append(Entry{
		Level:   Info,
		Source:  "下载",
		Message: strings.Repeat("🎉", 30) + "ação",
	})

	content := m.View().Content
	assertLineWidths(t, content, 20)
	assertValidUTF8(t, content)
	if !strings.Contains(content, "…") {
		t.Errorf("View() = %q, deveria truncar com reticências", content)
	}
}

func TestViewStripsControlCharacters(t *testing.T) {
	m := New()
	m.SetSize(60, 6)
	m.Append(Entry{Message: "linha 1\nlinha 2\tfim\x1b[31mvermelho\x1b[0m"})

	content := m.View().Content
	if got := strings.Count(content, "\n"); got != 2 {
		t.Errorf("View() tem %d quebras de linha, quer 2 (status, separador)", got)
	}
	if !strings.Contains(content, "linha 1 linha 2 fimvermelho") {
		t.Errorf("View() = %q, a mensagem deveria ser neutralizada", content)
	}
}

func TestViewSourceIsTruncatedBeforeMessage(t *testing.T) {
	m := New()
	m.SetSize(40, 4)
	m.Append(Entry{
		Level:   Info,
		Source:  strings.Repeat("origem", 20),
		Message: "mensagem importante",
	})

	content := m.View().Content
	assertLineWidths(t, content, 40)
	if !strings.Contains(content, "mensagem") {
		t.Errorf("View() = %q, deveria reservar espaço para a mensagem", content)
	}
}

func TestSeparatorWidth(t *testing.T) {
	m := New()
	m.SetSize(17, 4)

	lines := strings.Split(m.View().Content, "\n")
	if len(lines) < 2 {
		t.Fatalf("View() = %q, esperava status e separador", m.View().Content)
	}
	if got := ansi.StringWidth(strings.TrimSpace(lines[1])); got != 17 {
		t.Errorf("separador tem %d células, quer 17", got)
	}
	assertLineWidths(t, m.View().Content, 17)
}
