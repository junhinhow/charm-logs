# charm-logs

Componente de visualização de logs para [Bubble Tea](https://github.com/charmbracelet/bubbletea): buffer circular, modo follow, filtros por nível e por texto, exportação em texto puro e ponte com `log/slog`.

> **[Read in English](README.md)**

## Instalação

```bash
go get github.com/junhinhow/charm-logs@latest
```

## Uso

```go
package main

import (
	"log"
	"log/slog"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/junhinhow/charm-logs"
)

type model struct {
	logs logs.Model
}

func newModel() model {
	viewer := logs.New(logs.WithMaxEntries(500))
	viewer.SetSize(80, 20)

	logger := slog.New(viewer.SlogHandler())
	logger.Info("aplicação iniciada", "versão", "1.0")

	viewer.Append(logs.Entry{
		Time:    time.Now(),
		Level:   logs.Info,
		Source:  "download",
		Message: "baixando arquivo",
	})

	return model{logs: viewer}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.logs.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	updated, cmd := m.logs.Update(msg)
	m.logs = updated.(logs.Model)
	return m, cmd
}

func (m model) View() tea.View { return m.logs.View() }

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		log.Fatal(err)
	}
}
```

Sem `SetSize` inicial, o visualizador usa 80x24. Tratar `tea.WindowSizeMsg`
mantém o tamanho sincronizado com redimensionamentos do terminal.

## Teclas

| Tecla | Ação |
|-------|------|
| `j` / `↓` | Seleciona a próxima entrada (mais recente) |
| `k` / `↑` | Seleciona a entrada anterior (mais antiga); desliga o follow |
| `g` | Vai para a entrada mais antiga |
| `G` | Vai para a entrada mais recente e liga o follow |
| `/` | Entra no modo de busca |
| `enter` | Confirma a busca |
| `esc` | Cancela a busca e restaura o filtro anterior |
| `backspace` | Apaga o último caractere durante a busca |
| `l` | Cicla o nível mínimo: debug → info → warn → error → debug |
| `f` | Alterna o follow |
| `c` | Limpa o buffer |
| `e` | Exporta as entradas visíveis para `logs.txt` no diretório atual |

Enquanto o modo de busca está ativo, teclas imprimíveis são texto: `j`, `k`,
`g`, `l` e os demais atalhos não são interpretados como navegação.

## API

| Símbolo | Descrição |
|---------|-----------|
| `Level`, `Debug`, `Info`, `Warn`, `Error` | Severidade, ordenada da menor para a maior |
| `Level.String`, `ParseLevel` | Conversão textual; aceita `debug`, `info`, `warn`, `warning`, `error` |
| `Entry` | `Time`, `Level`, `Source` e `Message` de uma linha |
| `New(opts ...Option)` | Cria o visualizador (padrões: 1000 entradas, Debug, follow ligado, 80x24) |
| `WithMaxEntries(n)` | Define a capacidade do buffer circular; valores abaixo de 10 viram 10 |
| `Model.Append(entries ...Entry)` | Adiciona entradas e devolve um comando que produz `AppendedMsg` |
| `Model.Appendf(level, format, args...)` | Formata a mensagem com o instante atual e a adiciona |
| `Model.SetSize(width, height)` | Define o tamanho da área de visualização |
| `Model.SetMinLevel(level)` | Oculta entradas abaixo do nível apenas na renderização |
| `Model.SetFollow(bool)`, `Model.Follow()` | Controla e lê o modo follow |
| `Model.SetFilter(string)`, `Model.Filter()` | Controla e lê o filtro textual |
| `Model.Entries()` | Cópia das entradas visíveis (após filtros de nível e texto) |
| `Model.Len()` | Total de entradas no buffer, incluindo as ocultas |
| `Model.Clear()` | Esvazia o buffer |
| `Model.Export(io.Writer)` | Escreve as entradas visíveis em texto puro, sem ANSI |
| `Model.SlogHandler()` | Devolve um `slog.Handler` seguro para concorrência, ligado ao buffer |
| `Model.Init`, `Model.Update`, `Model.View` | Implementação de `tea.Model` |
| `AppendedMsg`, `ExportedMsg` | Mensagens produzidas pelos comandos de append e exportação |

## Notas de projeto

- **Buffer circular.** A capacidade padrão é 1000. Quando o buffer enche, a
  entrada mais antiga é descartada; o uso de memória não cresce com o fluxo de
  logs.
- **Follow.** O follow começa ligado. Novas entradas mantêm a mais recente
  selecionada. `k`/`↑` desliga o follow; `G` e `f` ligam novamente.
- **Seleção estável.** A entrada selecionada é rastreada por um número de
  sequência interno, não por um índice de slice. Assim, novas entradas não
  deslocam a seleção com o follow desligado, e entradas descartadas pelo buffer
  não deixam um índice inválido.
- **Filtragem.** O filtro textual é uma substring sem diferenciar maiúsculas de
  minúsculas, aplicada a `Message` e a `Source`. `SetMinLevel` afeta apenas a
  renderização; as entradas continuam no buffer e reaparecem se o nível for
  reduzido.
- **Layout.** Cada linha visível é `HH:MM:SS NÍVEL [origem] mensagem`.
  Sequências ANSI e caracteres de controle são neutralizados, e cada linha é
  truncada com `ansi.Truncate`, de modo que `View` nunca excede a largura
  configurada. A view também nunca excede a altura configurada.
- **Exportação.** `Export` escreve timestamps RFC 3339 e níveis em maiúsculas:
  `2006-01-02T15:04:05Z07:00 LEVEL [source] message`. Mensagens são achatadas
  em uma única linha. A tecla `e` usa o mesmo formato para escrever `logs.txt`.
- **slog.** `SlogHandler` mapeia os níveis do slog para `Debug`, `Info`, `Warn`
  e `Error`, anexa atributos como `key=value` e expande grupos com pontos
  (`req.user.name=ana`). `Handle` aceita todos os níveis; use `SetMinLevel` para
  ocultar entradas na interface.

## Limitações

- As mensagens são renderizadas em uma linha; não há quebra nem expansão de
  entrada.
- Sem suporte a mouse, seleção de intervalos arbitrários ou área de
  transferência.
- A paleta de cores é fixa e não é personalizável.
- `SlogHandler` não dispara uma repintura do Bubble Tea sozinho. Se registros
  chegarem fora do laço `Update`, envie uma mensagem ao programa (ou devolva o
  comando de `Model.Append` a partir de `Update`) para atualizar a view.
- `Append` e `SlogHandler` são seguros para uso concorrente porque apenas o
  buffer é protegido por mutex. Métodos que alteram o estado de interface
  (`SetSize`, `SetFilter`, `SetMinLevel`, `Clear`, `Update`, ...) devem ser
  chamados no laço de atualização do Bubble Tea.

## Licença

MIT
