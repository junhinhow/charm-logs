// Package logs fornece um visualizador de logs para aplicações Bubble Tea.
//
// O pacote cobre a lacuna entre formatar e visualizar: enquanto
// charm.land/log formata registros, este pacote mantém um buffer circular,
// renderiza as entradas em uma área com rolagem, filtra por nível e por texto
// e acompanha o fim do log (follow). A integração com log/slog é feita por
// SlogHandler.
//
// O Model implementa tea.Model. O aplicativo deve encaminhar os eventos de
// teclado para Update e chamar SetSize (ou deixar o Update tratar
// tea.WindowSizeMsg) antes de esperar conteúdo em View.
package logs

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	// defaultMaxEntries é a capacidade padrão do buffer circular.
	defaultMaxEntries = 1000
	// minMaxEntries é a menor capacidade aceita por WithMaxEntries.
	minMaxEntries = 10
	// exportFileName é o arquivo gerado pela tecla "e".
	exportFileName = "logs.txt"
	// defaultWidth e defaultHeight dão um tamanho utilizável ao Model antes
	// do primeiro tea.WindowSizeMsg.
	defaultWidth  = 80
	defaultHeight = 24
)

// Entry representa uma linha de log exibida pelo visualizador.
type Entry struct {
	// Time é o instante do evento. É exibido como HH:MM:SS em View e em
	// RFC 3339 em Export.
	Time time.Time
	// Level é a severidade do evento.
	Level Level
	// Source identifica a origem do evento, por exemplo "download". É
	// opcional.
	Source string
	// Message é o texto do evento. Caracteres de controle e sequências ANSI
	// são neutralizados na renderização para preservar o layout.
	Message string
}

// Option configura o Model criado por New.
type Option func(*options)

// options agrupa os valores configuráveis antes da construção do Model.
type options struct {
	maxEntries int
}

// WithMaxEntries define a capacidade do buffer circular.
//
// O padrão é 1000. Valores menores que 10 são elevados a 10, de modo que o
// buffer nunca deixa de ser útil. Ao exceder a capacidade, as entradas mais
// antigas são descartadas.
func WithMaxEntries(n int) Option {
	return func(o *options) {
		if n < minMaxEntries {
			n = minMaxEntries
		}
		o.maxEntries = n
	}
}

// AppendedMsg é devolvida pelo comando retornado por Append e Appendf.
//
// Ela serve apenas para acionar uma nova renderização; o aplicativo pode
// tratá-la como uma notificação e ignorá-la.
type AppendedMsg struct {
	// Count é a quantidade de entradas adicionadas.
	Count int
}

// ExportedMsg é devolvida pelo comando disparado pela tecla "e".
type ExportedMsg struct {
	// Path é o caminho do arquivo escrito.
	Path string
	// Err é o erro da escrita, ou nil em caso de sucesso.
	Err error
}

// Model é o visualizador de logs, pronto para uso como tea.Model.
//
// O Model deve ser tratado como um valor: Update devolve uma cópia com o
// estado de interface atualizado. O buffer é compartilhado entre as cópias,
// o que permite que Append e SlogHandler sejam chamados de outras goroutines.
// Os demais métodos de escrita (SetSize, SetFilter, SetMinLevel, Clear, etc.)
// não são seguros para uso concorrente; chame-os no laço do Bubble Tea.
type Model struct {
	buf *buffer

	width  int
	height int

	minLevel Level
	follow   bool
	filter   string

	// cursorSeq identifica a entrada selecionada por um número de sequência
	// estável. Diferente de um índice, ele não muda quando novas entradas
	// chegam ou quando o buffer circular descarta as mais antigas.
	// Zero significa "sem seleção explícita".
	cursorSeq uint64

	searching      bool
	previousFilter string
}

// Garante em tempo de compilação que Model satisfaz tea.Model.
var _ tea.Model = Model{}

// positioned associa uma entrada ao seu número de sequência.
type positioned struct {
	seq uint64
	Entry
}

// New cria um visualizador vazio.
//
// O buffer começa com 1000 entradas, o nível mínimo é Debug, o follow vem
// ligado e o tamanho inicial é 80x24 até que SetSize ou um
// tea.WindowSizeMsg o altere.
func New(opts ...Option) Model {
	cfg := options{maxEntries: defaultMaxEntries}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return Model{
		buf:      newBuffer(cfg.maxEntries),
		width:    defaultWidth,
		height:   defaultHeight,
		minLevel: Debug,
		follow:   true,
	}
}

// ensureBuffer devolve o buffer, criando-o quando o Model é o valor zero.
func (m *Model) ensureBuffer() *buffer {
	if m.buf == nil {
		m.buf = newBuffer(defaultMaxEntries)
	}
	return m.buf
}

// Append adiciona entradas ao buffer, respeitando a capacidade circular.
//
// O comando devolvido produz AppendedMsg quando ao menos uma entrada foi
// adicionada. Devolve nil quando a lista é vazia. É seguro chamar Append de
// outras goroutines, desde que o estado de interface não seja tocado.
func (m *Model) Append(entries ...Entry) tea.Cmd {
	buf := m.ensureBuffer()
	for _, e := range entries {
		buf.add(e)
	}

	if len(entries) == 0 {
		return nil
	}

	count := len(entries)
	return func() tea.Msg {
		return AppendedMsg{Count: count}
	}
}

// Appendf formata a mensagem e adiciona uma entrada com o instante atual.
func (m *Model) Appendf(level Level, format string, args ...any) tea.Cmd {
	return m.Append(Entry{
		Time:    time.Now(),
		Level:   level,
		Message: fmt.Sprintf(format, args...),
	})
}

// SetSize define a largura e a altura disponíveis para o visualizador.
//
// Valores negativos são tratados como zero. Com largura ou altura zero, View
// devolve uma view vazia.
func (m *Model) SetSize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	m.width = width
	m.height = height
}

// SetMinLevel define o nível mínimo exibido.
//
// O filtro acontece apenas na renderização: as entradas continuam no buffer.
// Valores fora do intervalo conhecido são limitados a Debug ou Error.
func (m *Model) SetMinLevel(l Level) {
	m.minLevel = clampLevel(l)
}

// SetFollow liga ou desliga o acompanhamento do fim do log.
//
// Ao ligar, a seleção volta para a entrada mais recente. Ao desligar, a
// seleção permanece na entrada mais recente e passa a acompanhar apenas a
// navegação manual.
func (m *Model) SetFollow(b bool) {
	m.follow = b
	if vis := m.visible(); len(vis) > 0 {
		m.cursorSeq = vis[len(vis)-1].seq
	} else {
		m.cursorSeq = 0
	}
}

// Follow informa se o visualizador está acompanhando o fim do log.
func (m Model) Follow() bool {
	return m.follow
}

// SetFilter define o texto de busca.
//
// A busca é uma substring sem diferenciar maiúsculas de minúsculas,
// aplicada à mensagem e à origem. Caracteres de controle são neutralizados
// para não quebrar a linha de status.
func (m *Model) SetFilter(s string) {
	m.filter = sanitizeText(s)
	m.searching = false
	m.previousFilter = ""
}

// Filter devolve o texto de busca atual.
func (m Model) Filter() string {
	return m.filter
}

// Entries devolve uma cópia das entradas visíveis, na ordem de inserção.
//
// A visibilidade considera o nível mínimo e o filtro textual. Alterar o slice
// devolvido não afeta o Model.
func (m Model) Entries() []Entry {
	vis := m.visible()
	out := make([]Entry, len(vis))
	for i, p := range vis {
		out[i] = p.Entry
	}
	return out
}

// visible devolve as entradas que passam pelo nível mínimo e pelo filtro,
// acompanhadas do número de sequência.
func (m Model) visible() []positioned {
	all, first := m.snapshot()
	if len(all) == 0 {
		return nil
	}

	query := strings.ToLower(m.filter)
	out := make([]positioned, 0, len(all))
	for i, e := range all {
		if e.Level < m.minLevel {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(e.Message), query) &&
			!strings.Contains(strings.ToLower(e.Source), query) {
			continue
		}
		out = append(out, positioned{seq: first + uint64(i), Entry: e})
	}
	return out
}

// Len devolve a quantidade total de entradas no buffer, incluindo as que
// estão ocultas por filtro ou por nível.
func (m Model) Len() int {
	if m.buf == nil {
		return 0
	}
	return m.buf.len()
}

// Clear remove todas as entradas do buffer.
//
// Filtro, nível mínimo e follow são preservados.
func (m *Model) Clear() {
	if m.buf != nil {
		m.buf.clear()
	}
	m.cursorSeq = 0
	m.searching = false
	m.previousFilter = ""
}

// snapshot devolve uma cópia ordenada do buffer e a sequência da primeira
// entrada.
func (m Model) snapshot() ([]Entry, uint64) {
	if m.buf == nil {
		return nil, 0
	}
	return m.buf.snapshot()
}

// selectedIndex devolve o índice da entrada selecionada em vis.
//
// Quando não há seleção explícita, ou quando a entrada selecionada saiu da
// lista (por filtro ou descarte), a escolha recai sobre a primeira entrada
// mais recente que ela; se não houver, sobre a última.
func (m Model) selectedIndex(vis []positioned) int {
	if len(vis) == 0 {
		return 0
	}
	if m.follow || m.cursorSeq == 0 {
		return len(vis) - 1
	}

	i := sort.Search(len(vis), func(i int) bool {
		return vis[i].seq >= m.cursorSeq
	})
	if i >= len(vis) {
		return len(vis) - 1
	}
	return i
}

// Init implementa tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implementa tea.Model.
//
// Trata tea.WindowSizeMsg, tea.KeyPressMsg e AppendedMsg. Demais mensagens
// são ignoradas.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

// updateKey processa um evento de teclado.
func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.updateSearchKey(msg)
	}

	switch msg.String() {
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "g":
		m.goTop()
	case "G":
		m.goBottom()
	case "/":
		m.startSearch()
	case "l":
		m.cycleLevel()
	case "f":
		m.SetFollow(!m.follow)
	case "c":
		m.Clear()
	case "e":
		return m, m.exportCmd()
	}
	return m, nil
}

// updateSearchKey processa as teclas do modo de busca.
func (m Model) updateSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEscape:
		m.filter = m.previousFilter
		m.previousFilter = ""
		m.searching = false
		return m, nil
	case tea.KeyEnter:
		m.previousFilter = ""
		m.searching = false
		return m, nil
	case tea.KeyBackspace:
		m.filter = dropLastRune(m.filter)
		return m, nil
	}

	if msg.Text != "" {
		m.filter += sanitizeText(msg.Text)
	}
	return m, nil
}

// moveUp seleciona a entrada anterior e desliga o follow.
func (m *Model) moveUp() {
	vis := m.visible()
	if len(vis) == 0 {
		m.follow = false
		m.cursorSeq = 0
		return
	}

	idx := len(vis) - 1
	if !m.follow {
		idx = m.selectedIndex(vis)
	}
	m.follow = false
	if idx > 0 {
		idx--
	}
	m.cursorSeq = vis[idx].seq
}

// moveDown seleciona a entrada seguinte, sem reativar o follow.
func (m *Model) moveDown() {
	if m.follow {
		return
	}

	vis := m.visible()
	if len(vis) == 0 {
		m.cursorSeq = 0
		return
	}

	idx := m.selectedIndex(vis)
	if idx < len(vis)-1 {
		idx++
	}
	m.cursorSeq = vis[idx].seq
}

// goTop seleciona a entrada mais antiga.
func (m *Model) goTop() {
	m.follow = false
	vis := m.visible()
	if len(vis) == 0 {
		m.cursorSeq = 0
		return
	}
	m.cursorSeq = vis[0].seq
}

// goBottom seleciona a entrada mais recente e reativa o follow.
func (m *Model) goBottom() {
	m.SetFollow(true)
}

// startSearch entra no modo de busca com uma consulta vazia.
func (m *Model) startSearch() {
	m.previousFilter = m.filter
	m.filter = ""
	m.searching = true
}

// cycleLevel avança para o próximo nível mínimo, voltando a Debug depois de
// Error.
func (m *Model) cycleLevel() {
	switch m.minLevel {
	case Debug:
		m.minLevel = Info
	case Info:
		m.minLevel = Warn
	case Warn:
		m.minLevel = Error
	default:
		m.minLevel = Debug
	}
}

// exportCmd cria o comando que escreve as entradas visíveis em logs.txt.
func (m Model) exportCmd() tea.Cmd {
	entries := m.Entries()
	return func() tea.Msg {
		return ExportedMsg{Path: exportFileName, Err: writeFile(exportFileName, entries)}
	}
}

// dropLastRune remove o último caractere da string, respeitando UTF-8.
func dropLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}
