package acceptance_test

import (
	"bytes"
	"sync"
)

type LogBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (b *LogBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.Write(data)
}

func (b *LogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.String()
}
