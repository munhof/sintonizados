package memory

import (
	"context"
	"github.com/munhof/sintonizados/internal/domain"
	"sort"
	"sync"
)

type SessionStore struct {
	mu   sync.RWMutex
	data map[string]domain.Snapshot
}

func NewSessionStore() *SessionStore { return &SessionStore{data: make(map[string]domain.Snapshot)} }
func (s *SessionStore) Create(v domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[v.ID]; ok {
		return domain.ErrConflict
	}
	s.data[v.ID] = domain.Snapshot{Session: v, Subtitles: []domain.Subtitle{}}
	return nil
}
func clone(v domain.Snapshot) domain.Snapshot {
	v.Subtitles = append([]domain.Subtitle{}, v.Subtitles...)
	k := v.Session.Knowledge
	k.RecentTranscript = append([]string{}, k.RecentTranscript...)
	k.Entities = append([]string{}, k.Entities...)
	copyMap := func(m map[string]string) map[string]string {
		r := map[string]string{}
		for k, v := range m {
			r[k] = v
		}
		return r
	}
	k.Terminology = copyMap(k.Terminology)
	k.Glossary = copyMap(k.Glossary)
	k.Metadata = copyMap(k.Metadata)
	v.Session.Knowledge = k
	return v
}
func (s *SessionStore) Get(id string) (domain.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[id]
	if !ok {
		return v, domain.ErrNotFound
	}
	return clone(v), nil
}
func (s *SessionStore) List() []domain.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := []domain.Session{}
	for _, x := range s.data {
		v = append(v, clone(x).Session)
	}
	sort.Slice(v, func(i, j int) bool { return v[i].ID < v[j].ID })
	return v
}
func (s *SessionStore) Update(id string, f func(*domain.Snapshot)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	f(&v)
	s.data[id] = v
	return nil
}

type subscriber struct {
	session  string
	kind     domain.EventKind
	reliable bool
	ch       chan domain.Event
	done     chan struct{}
}
type EventBus struct {
	mu   sync.RWMutex
	next int
	subs map[int]*subscriber
}

func NewEventBus() *EventBus { return &EventBus{subs: map[int]*subscriber{}} }
func (b *EventBus) Subscribe(session string, kind domain.EventKind, reliable ...bool) (<-chan domain.Event, func()) {
	b.mu.Lock()
	b.next++
	id := b.next
	s := &subscriber{session: session, kind: kind, ch: make(chan domain.Event, 32), done: make(chan struct{})}
	if len(reliable) > 0 {
		s.reliable = reliable[0]
	}
	b.subs[id] = s
	b.mu.Unlock()
	var once sync.Once
	return s.ch, func() { once.Do(func() { b.mu.Lock(); delete(b.subs, id); close(s.done); b.mu.Unlock() }) }
}
func (b *EventBus) Publish(ctx context.Context, e domain.Event) error {
	b.mu.RLock()
	subs := []*subscriber{}
	for _, s := range b.subs {
		if s.session == e.SessionID && (s.kind == "" || s.kind == e.Kind) {
			subs = append(subs, s)
		}
	}
	b.mu.RUnlock()
	for _, s := range subs {
		if s.reliable {
			select {
			case s.ch <- e:
			case <-s.done:
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			select {
			case <-s.done:
			case s.ch <- e:
			default:
			}
		}
	}
	return nil
}
