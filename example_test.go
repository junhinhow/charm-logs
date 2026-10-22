package logs_test

import (
	"fmt"
	"os"
	"time"

	"github.com/junhinhow/charm-logs"
)

// ExampleModel_Export mostra a exportação em texto puro.
func ExampleModel_Export() {
	m := logs.New()
	m.Append(
		logs.Entry{
			Time:    time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC),
			Level:   logs.Info,
			Source:  "download",
			Message: "arquivo concluído",
		},
		logs.Entry{
			Time:    time.Date(2026, 10, 22, 12, 0, 1, 0, time.UTC),
			Level:   logs.Error,
			Message: "falha na conexão",
		},
	)

	if err := m.Export(os.Stdout); err != nil {
		fmt.Println(err)
	}

	// Output:
	// 2026-10-22T12:00:00Z INFO [download] arquivo concluído
	// 2026-10-22T12:00:01Z ERROR falha na conexão
}

// ExampleParseLevel mostra a conversão de um nome de nível.
func ExampleParseLevel() {
	level, err := logs.ParseLevel("warning")
	fmt.Println(level, err)

	// Output:
	// warn <nil>
}
