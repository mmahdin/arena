package embedding

import (
	"container/list"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"time"
)

// cache is owned by Service.mu. Neither keys nor entries retain source text.
type entry struct {
	key     key
	vector  []float32
	model   string
	info    trace.EmbeddingInfo
	expires time.Time
}

type cache struct {
	entries map[key]*list.Element
	lru     *list.List
}

func newCache() cache                   { return cache{entries: map[key]*list.Element{}, lru: list.New()} }
func (c *cache) remove(e *list.Element) { delete(c.entries, e.Value.(entry).key); c.lru.Remove(e) }
func (c *cache) get(k key, now time.Time) (ports.EmbeddingResponse, trace.EmbeddingInfo, bool) {
	e := c.entries[k]
	if e == nil {
		return ports.EmbeddingResponse{}, trace.EmbeddingInfo{}, false
	}
	item := e.Value.(entry)
	if !now.Before(item.expires) {
		c.remove(e)
		return ports.EmbeddingResponse{}, trace.EmbeddingInfo{}, false
	}
	c.lru.MoveToFront(e)
	return ports.EmbeddingResponse{Model: item.model, Vector: append([]float32(nil), item.vector...)}, item.info, true
}
func (c *cache) put(k key, r ports.EmbeddingResponse, info trace.EmbeddingInfo, now time.Time, config Config) {
	// Expired entries must not displace live entries just because they were not read.
	for e := c.lru.Back(); e != nil; {
		previous := e.Prev()
		if !now.Before(e.Value.(entry).expires) {
			c.remove(e)
		}
		e = previous
	}
	if e := c.entries[k]; e != nil {
		c.remove(e)
	}
	e := c.lru.PushFront(entry{key: k, vector: append([]float32(nil), r.Vector...), model: r.Model, info: info, expires: now.Add(time.Duration(config.TTLSeconds) * time.Second)})
	c.entries[k] = e
	for c.lru.Len() > config.MaxEntries {
		c.remove(c.lru.Back())
	}
}
