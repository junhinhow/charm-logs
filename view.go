package logs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	// styleStatus é o estilo da linha de status.
	styleStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Bold(true)
	// styleSeparator é o estilo do separador horizontal.
	styleSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3F3F46"))
	// styleTimestamp é o estilo do horário.
	styleTimestamp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
	// styleSource é o estilo da origem.
	styleSource = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA"))
	// styleSelected é o estilo do marcador da entrada selecionada.
	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)
	// stylePlaceholder é o estilo do aviso de lista vazia.
	stylePlaceholder = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))
)

// levelStyle devolve o estilo de cor do nível.
func levelStyle(l Level) lipgloss.Style {
	switch l {
	case Debug:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	case Info:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#61AFEF"))
	case Warn:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
	case Error:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	}
}

// View implementa tea.Model.
//
// Devolve uma view vazia quando a largura ou a altura é zero. A largura de
// cada linha nunca excede m.width, e a quantidade de linhas nunca excede
// m.height.
func (m Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return tea.NewView("")
	}

	lines := make([]string, 0, m.height)
	lines = append(lines, m.statusLine())
	if m.height >= 2 {
		lines = append(lines, m.separatorLine())
	}

	area := m.height - len(lines)
	if area > 0 {
		entries := m.visible()
		if len(entries) == 0 {
			lines = append(lines, ansi.Truncate(stylePlaceholder.Render("(sem entradas)"), m.width, "…"))
		} else {
			selected := m.selectedIndex(entries)
			start, end := viewport(selected, len(entries), area)
			for i := start; i < end; i++ {
				lines = append(lines, m.entryLine(entries[i].Entry, i == selected))
			}
		}
	}

	return tea.NewView(strings.Join(lines, "\n"))
}

// statusLine monta a linha de status com contagem, nível, follow e filtro.
func (m Model) statusLine() string {
	count := m.Len()
	noun := "entradas"
	if count == 1 {
		noun = "entrada"
	}

	follow := "follow off"
	if m.follow {
		follow = "follow on"
	}

	filter := "-"
	if m.filter != "" {
		filter = m.filter
	}

	status := fmt.Sprintf("%d %s · %s · %s · filtro: %s",
		count, noun, m.minLevel.String(), follow, filter)
	if m.searching {
		status += "▌"
	}

	return ansi.Truncate(styleStatus.Render(status), m.width, "…")
}

// separatorLine monta o separador horizontal com exatamente m.width células.
func (m Model) separatorLine() string {
	if m.width <= 0 {
		return ""
	}
	return styleSeparator.Render(strings.Repeat("─", m.width))
}

// viewport devolve o intervalo [start, end) de entradas que cabe na área.
//
// A seleção é mantida visível e, quando possível, na última linha da janela;
// perto do início, a janela começa na primeira entrada e a seleção sobe.
func viewport(selected, n, area int) (int, int) {
	if n <= 0 || area <= 0 {
		return 0, 0
	}

	if selected < 0 {
		selected = 0
	}
	if selected > n-1 {
		selected = n - 1
	}

	end := selected + 1
	if end < area {
		end = min(n, area)
	}

	start := end - area
	if start < 0 {
		start = 0
	}
	return start, end
}

// entryLine renderiza uma entrada com marcador, horário, nível, origem e
// mensagem, truncando o conjunto para não exceder m.width.
func (m Model) entryLine(e Entry, selected bool) string {
	if m.width <= 0 {
		return ""
	}

	marker := "  "
	if selected {
		marker = styleSelected.Render("▸") + " "
	}

	timestamp := styleTimestamp.Render(e.Time.Format("15:04:05"))
	level := levelStyle(e.Level).Render(e.Level.short())

	prefix := marker + timestamp + " " + level + " "
	remaining := m.width - ansi.StringWidth(prefix)
	if remaining < 0 {
		remaining = 0
	}

	// A origem só entra quando sobra espaço para uma mensagem legível; caso
	// contrário, a mensagem tem prioridade.
	const minMessageWidth = 10

	source := ""
	sourceText := ""
	if e.Source != "" {
		sourceText = "[" + sanitizeText(e.Source) + "] "
	}
	if sourceText != "" && remaining >= minMessageWidth+4 {
		budget := remaining - minMessageWidth
		if ansi.StringWidth(sourceText) > budget {
			sourceText = ansi.Truncate(sourceText, budget, "…")
		}
		source = styleSource.Render(sourceText)
		remaining -= ansi.StringWidth(sourceText)
	}

	if remaining < 0 {
		remaining = 0
	}
	message := ansi.Truncate(sanitizeText(e.Message), remaining, "…")

	return ansi.Truncate(prefix+source+message, m.width, "…")
}

// sanitizeText remove sequências ANSI e troca caracteres de controle por
// espaços, preservando a validade UTF-8 e o layout.
func sanitizeText(s string) string {
	s = ansi.Strip(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			// Troca controles por espaço para não colar palavras nem
			// quebrar a linha.
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
