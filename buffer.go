package logs

import "sync"

// buffer é um buffer circular de entradas de log.
//
// O mutex protege apenas o buffer: o estado de interface do Model continua
// sendo manipulado pelo laço single-threaded do Bubble Tea. Essa separação
// permite que um slog.Handler concorrente chame Append sem corrida de dados.
type buffer struct {
	mu    sync.Mutex
	ring  []Entry
	start int
	count int
	// nextSeq é o número de sequência atribuído à última entrada. As
	// sequências são contíguas, o que permite deduzir a sequência da
	// entrada mais antiga a partir de nextSeq e count.
	nextSeq uint64
}

// newBuffer cria um buffer com a capacidade informada.
func newBuffer(capacity int) *buffer {
	if capacity < minMaxEntries {
		capacity = minMaxEntries
	}
	return &buffer{ring: make([]Entry, capacity)}
}

// add insere uma entrada, descartando a mais antiga quando o buffer está
// cheio.
func (b *buffer) add(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.ring) == 0 {
		return
	}

	b.nextSeq++

	if b.count == len(b.ring) {
		b.ring[b.start] = e
		b.start = (b.start + 1) % len(b.ring)
		return
	}

	b.ring[(b.start+b.count)%len(b.ring)] = e
	b.count++
}

// snapshot devolve uma cópia das entradas na ordem de inserção e a sequência
// da primeira delas.
func (b *buffer) snapshot() ([]Entry, uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil, 0
	}

	out := make([]Entry, b.count)
	for i := range out {
		out[i] = b.ring[(b.start+i)%len(b.ring)]
	}
	return out, b.nextSeq - uint64(b.count) + 1
}

// len devolve a quantidade de entradas armazenadas.
func (b *buffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// clear esvazia o buffer e libera as referências às entradas antigas.
func (b *buffer) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	clear(b.ring)
	b.start = 0
	b.count = 0
	b.nextSeq = 0
}
