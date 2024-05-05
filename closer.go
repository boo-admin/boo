package boo

import (
	"io"
	"sync"
	
	"github.com/boo-admin/boo/client"
)


type CloseFunc = client.CloseFunc

func NoCloser() io.Closer {
	return client.NoCloser()
}

type SyncCloser struct {
	closer io.Closer
	mu     sync.Mutex
}

func (sc *SyncCloser) Set(closer io.Closer) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.closer = closer
}

func (sc *SyncCloser) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closer != nil {
		return sc.closer.Close()
	}
	return nil
}
