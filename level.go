package logs

import (
	"fmt"
	"strings"
)

// Level representa o nível de severidade de uma entrada de log.
//
// A ordem é crescente: Debug < Info < Warn < Error. O nível mínimo do
// visualizador usa essa ordem para filtrar a renderização sem descartar
// entradas do buffer.
type Level int

const (
	// Debug indica informação detalhada, útil durante o desenvolvimento.
	Debug Level = iota
	// Info indica um evento normal da aplicação.
	Info
	// Warn indica uma situação inesperada que não interrompe a execução.
	Warn
	// Error indica uma falha.
	Error
)

// String devolve o nome do nível em minúsculas.
//
// Níveis fora do intervalo conhecido devolvem "unknown".
func (l Level) String() string {
	switch l {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLevel converte um nome de nível em Level.
//
// Aceita "debug", "info", "warn", "warning" e "error", sem diferenciar
// maiúsculas de minúsculas e ignorando espaços nas extremidades. Em caso de
// erro, devolve Info juntamente com o erro.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "warn", "warning":
		return Warn, nil
	case "error":
		return Error, nil
	default:
		return Info, fmt.Errorf("logs: nível inválido %q", s)
	}
}

// valid informa se o nível está no intervalo conhecido.
func (l Level) valid() bool {
	return l >= Debug && l <= Error
}

// clampLevel limita um nível arbitrário ao intervalo conhecido.
func clampLevel(l Level) Level {
	if l < Debug {
		return Debug
	}
	if l > Error {
		return Error
	}
	return l
}

// short devolve o rótulo de três letras usado na renderização.
func (l Level) short() string {
	switch l {
	case Debug:
		return "DBG"
	case Info:
		return "INF"
	case Warn:
		return "WRN"
	case Error:
		return "ERR"
	default:
		return "???"
	}
}
