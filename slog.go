package logs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// SlogHandler devolve um slog.Handler que converte registros em Entry e os
// adiciona ao buffer do Model.
//
// Os níveis do slog são mapeados para Debug, Info, Warn e Error. Atributos
// são anexados à mensagem no formato chave=valor, na ordem em que foram
// declarados; grupos viram prefixos separados por ponto. O horário do
// registro é preservado.
//
// O handler é seguro para uso concorrente: o único estado compartilhado é o
// buffer, protegido por mutex. WithAttrs e WithGroup devolvem novos handlers
// sem alterar o original.
func (m *Model) SlogHandler() slog.Handler {
	if m == nil {
		return &slogHandler{}
	}
	return &slogHandler{buf: m.ensureBuffer()}
}

// slogHandler implementa slog.Handler sobre um buffer compartilhado.
type slogHandler struct {
	buf *buffer
	// attrs guarda os atributos fixados por WithAttrs já formatados com os
	// grupos vigentes naquele momento. Isso preserva a semântica do slog, em
	// que um grupo criado depois não reescreve atributos anteriores.
	attrs  []string
	groups []string
}

// Garante em tempo de compilação que slogHandler satisfaz slog.Handler.
var _ slog.Handler = (*slogHandler)(nil)

// Enabled informa que todos os níveis são aceitos.
//
// O filtro por nível é responsabilidade do visualizador (SetMinLevel), não do
// handler: assim as entradas continuam disponíveis no buffer mesmo quando
// ocultas na tela.
func (h *slogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle converte o registro e o adiciona ao buffer.
func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	if h.buf == nil {
		return nil
	}

	message := r.Message

	fields := make([]string, 0, len(h.attrs)+r.NumAttrs())
	fields = append(fields, h.attrs...)

	var recordAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		recordAttrs = append(recordAttrs, a)
		return true
	})
	fields = append(fields, formatAttrs(h.groups, recordAttrs)...)

	if len(fields) > 0 {
		var b strings.Builder
		b.WriteString(message)
		for _, field := range fields {
			b.WriteByte(' ')
			b.WriteString(field)
		}
		message = b.String()
	}

	when := r.Time
	if when.IsZero() {
		when = time.Now()
	}

	h.buf.add(Entry{
		Time:    when,
		Level:   slogLevel(r.Level),
		Message: message,
	})
	return nil
}

// WithAttrs devolve um handler que acrescenta os atributos informados.
func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	clone := *h
	clone.attrs = make([]string, 0, len(h.attrs)+len(attrs))
	clone.attrs = append(clone.attrs, h.attrs...)
	clone.attrs = append(clone.attrs, formatAttrs(h.groups, attrs)...)
	return &clone
}

// WithGroup devolve um handler que prefixa os próximos atributos com o nome
// do grupo.
func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := *h
	clone.groups = make([]string, 0, len(h.groups)+1)
	clone.groups = append(clone.groups, h.groups...)
	clone.groups = append(clone.groups, name)
	return &clone
}

// slogLevel converte um nível de log/slog em Level.
func slogLevel(l slog.Level) Level {
	switch {
	case l < slog.LevelInfo:
		return Debug
	case l < slog.LevelWarn:
		return Info
	case l < slog.LevelError:
		return Warn
	default:
		return Error
	}
}

// formatAttrs serializa atributos em campos chave=valor.
//
// A ordem é determinística: primeiro os atributos fixados por WithAttrs,
// depois os do registro, ambos na ordem de declaração. Grupos são expandidos
// recursivamente.
func formatAttrs(groups []string, attrs []slog.Attr) []string {
	var fields []string

	var walk func(prefix []string, a slog.Attr)
	walk = func(prefix []string, a slog.Attr) {
		a.Value = a.Value.Resolve()

		if a.Value.Kind() == slog.KindGroup {
			next := prefix
			if a.Key != "" {
				next = appendKey(prefix, a.Key)
			}
			for _, child := range a.Value.Group() {
				walk(next, child)
			}
			return
		}

		if a.Key == "" {
			return
		}

		key := strings.Join(appendKey(prefix, a.Key), ".")
		fields = append(fields, key+"="+formatValue(a.Value))
	}

	for _, a := range attrs {
		walk(groups, a)
	}
	return fields
}

// appendKey devolve uma cópia do prefixo com a chave acrescentada, sem
// compartilhar o array subjacente.
func appendKey(prefix []string, key string) []string {
	out := make([]string, len(prefix)+1)
	copy(out, prefix)
	out[len(prefix)] = key
	return out
}

// formatValue devolve a representação textual de um valor de atributo.
func formatValue(v slog.Value) string {
	if v.Kind() == slog.KindString {
		return sanitizeText(v.String())
	}
	return sanitizeText(fmt.Sprint(v.Any()))
}
