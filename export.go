package logs

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

// Export escreve as entradas visíveis em texto puro, sem sequências ANSI.
//
// Cada entrada ocupa uma linha no formato
// "2006-01-02T15:04:05Z07:00 LEVEL [source] message". O bloco [source] é
// omitido quando a origem é vazia. Quebras de linha e outros caracteres de
// controle da mensagem são convertidos em espaços para manter uma entrada por
// linha.
//
// Devolve erro quando w é nil ou quando a escrita falha.
func (m Model) Export(w io.Writer) error {
	if w == nil {
		return errors.New("logs: writer nulo")
	}
	return writeEntries(w, m.Entries())
}

// writeEntries serializa as entradas no formato de Export.
func writeEntries(w io.Writer, entries []Entry) error {
	var b strings.Builder
	for _, e := range entries {
		b.Reset()
		b.WriteString(e.Time.Format(time.RFC3339))
		b.WriteByte(' ')
		b.WriteString(strings.ToUpper(e.Level.String()))

		if source := sanitizeText(e.Source); source != "" {
			b.WriteString(" [")
			b.WriteString(source)
			b.WriteByte(']')
		}

		b.WriteByte(' ')
		b.WriteString(sanitizeText(e.Message))
		b.WriteByte('\n')

		if _, err := io.WriteString(w, b.String()); err != nil {
			return err
		}
	}
	return nil
}

// writeFile grava as entradas em um arquivo.
func writeFile(path string, entries []Entry) error {
	var b strings.Builder
	if err := writeEntries(&b, entries); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
