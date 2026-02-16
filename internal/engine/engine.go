package engine

import (
	"sync"

	"github.com/ailert/ailert/internal/pattern"
	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/types"
)

// Result is the outcome of processing one record.
type Result struct {
	Level    types.Level
	Hash     string
	Sample   string
	IsNew    bool
	Suppressed bool
	Count   int64
}

type levelHash struct {
	level types.Level
	hash  string
}

// Engine runs the pattern extraction and store lookup.
type Engine struct {
	mu       sync.RWMutex
	store    *store.Store
	patterns map[levelHash]*pattern.Pattern // (level, hash) -> pattern for WeakEqual within same level
}

// New returns an engine that uses the given store.
func New(st *store.Store) *Engine {
	return &Engine{
		store:   st,
		patterns: make(map[levelHash]*pattern.Pattern),
	}
}

// Process takes a record and returns the pattern result (hash, new/known, suppressed).
func (e *Engine) Process(r *types.Record) Result {
	level := r.Level
	if level == types.LevelUnknown {
		level = pattern.DetectLevel(r.Message)
	}
	pat := pattern.New(r.Message)
	hash := pat.Hash()

	if st := e.store; st.IsSuppressed(hash) {
		return Result{Level: level, Hash: hash, Sample: r.Message, IsNew: false, Suppressed: true, Count: 0}
	}

	e.mu.Lock()
	var existingHash string
	lh := levelHash{level: level, hash: hash}
	for k, p := range e.patterns {
		if k.level != level {
			continue
		}
		if p.WeakEqual(pat) {
			existingHash = k.hash
			break
		}
	}
	if existingHash != "" {
		hash = existingHash
	} else {
		e.patterns[lh] = pat
	}
	e.mu.Unlock()

	isNew := e.store.Seen(level, hash, r.Message)
	count := e.store.GetCount(level, hash)
	return Result{
		Level: level, Hash: hash, Sample: r.Message,
		IsNew: isNew, Suppressed: false, Count: count,
	}
}
