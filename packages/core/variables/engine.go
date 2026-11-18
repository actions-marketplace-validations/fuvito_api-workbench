package variables

import (
	crand "crypto/rand"
	"encoding/binary"
	"hash/fnv"
	"math/rand/v2"
	"time"
	_ "time/tzdata" // lets tz=Europe/Istanbul work on Windows and in minimal containers
)

// Engine generates the values behind {{$function}} calls. All randomness comes from one seeded
// generator, so a run can be replayed exactly with the same seed. An Engine is not safe for
// concurrent use: give each scenario its own with Derive.
type Engine struct {
	seed int64
	rng  *rand.Rand
	now  func() time.Time
	used *bool   // did this engine, or any engine derived from it, draw from the generator?
	ups  []*bool // the flags of the engines this one was derived from, marked too, so a run knows whether any scenario used randomness
}

// NewEngine returns an engine for the given seed that reads the system clock.
func NewEngine(seed int64) *Engine {
	used := false
	return &Engine{seed: seed, rng: newRNG(seed, 0), now: time.Now, used: &used}
}

func newRNG(seed int64, stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), stream))
}

// RandomSeed returns a fresh, unpredictable seed.
func RandomSeed() int64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return time.Now().UnixNano()
	}
	return int64(binary.LittleEndian.Uint64(b[:]) >> 1) // non-negative, easy to type and read
}

// WithClock returns a copy of the engine that reads the given clock (for tests).
func (e *Engine) WithClock(now func() time.Time) *Engine {
	c := *e
	c.now = now
	return &c
}

// Derive returns an engine with its own generator, determined by the run seed and key (for
// example a scenario's path). The values a scenario gets therefore do not depend on the order in
// which scenarios run, or on whether they run side by side.
func (e *Engine) Derive(key string) *Engine {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	c := *e
	c.rng = newRNG(e.seed, h.Sum64()|1)
	own := false
	c.used = &own
	c.ups = append(append([]*bool{}, e.ups...), e.used)
	return &c
}

// Seed is the seed the run started with.
func (e *Engine) Seed() int64 { return e.seed }

// UsedRandomness reports whether any value has been drawn from this engine's generator, or from
// the generator of an engine derived from it.
func (e *Engine) UsedRandomness() bool { return *e.used }

func (e *Engine) draw() *rand.Rand {
	*e.used = true
	for _, up := range e.ups {
		*up = true
	}
	return e.rng
}
