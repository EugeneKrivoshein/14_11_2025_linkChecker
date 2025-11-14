package worker_test

import (
	"14_11_2025_linkChecker/internal/util"
	"14_11_2025_linkChecker/internal/worker"
	"14_11_2025_linkChecker/models"
	"sync"
	"testing"
	"time"
)

type InMemoryStore struct {
	mu     sync.Mutex
	lastID int64
	sets   map[int64]*models.LinkSet
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{sets: make(map[int64]*models.LinkSet)}
}

func (s *InMemoryStore) CreateSet(links []string) (int64, *models.LinkSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastID++
	id := s.lastID
	set := &models.LinkSet{
		ID:      id,
		Links:   links,
		Results: make(map[string]*models.LinkResult),
		Status:  "processing",
	}
	s.sets[id] = set
	return id, set, nil
}

func (s *InMemoryStore) GetSet(id int64) (*models.LinkSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sets[id], nil
}

func (s *InMemoryStore) UpdateLinkResult(id int64, url string, res models.LinkResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	set, ok := s.sets[id]
	if !ok || set == nil {
		return nil
	}

	set.Results[url] = &res

	if len(set.Results) < len(set.Links) {
		set.Status = "processing"
		return nil
	}

	allDone := true
	for _, r := range set.Results {
		if r.State != models.StateAvailable && r.State != models.StateNotAvailable {
			allDone = false
			break
		}
	}

	if allDone {
		set.Status = "done"
	}
	return nil
}

func (s *InMemoryStore) ListUnfinished() ([]*models.LinkSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*models.LinkSet
	for _, set := range s.sets {
		if set.Status != "done" {
			out = append(out, set)
		}
	}
	return out, nil
}

func (s *InMemoryStore) ListSets(ids []int64) ([]*models.LinkSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []*models.LinkSet
	for _, id := range ids {
		set, ok := s.sets[id]
		if ok {
			out = append(out, set)
		}
	}
	return out, nil
}

func TestWorkerGracefulRestart(t *testing.T) {
	store := NewInMemoryStore()

	origCheck := util.CheckURL
	defer func() { util.CheckURL = origCheck }()

	util.CheckURL = func(url string) (bool, string) {
		switch url {
		case "http://link1.com":
			time.Sleep(300 * time.Millisecond)
			return true, "ok"
		case "http://link2.com":
			return false, "not available"
		case "http://link3.com":
			return true, "ok"
		default:
			return false, "not found"
		}
	}

	id1, _, _ := store.CreateSet([]string{"http://link1.com", "http://link2.com"})
	id2, _, _ := store.CreateSet([]string{"http://link3.com"})

	mgr := worker.NewManager(store, 1)
	go mgr.Run()

	mgr.Enqueue(id1)
	mgr.Enqueue(id2)

	time.Sleep(100 * time.Millisecond)

	mgr.Stop()

	unfinished, _ := store.ListUnfinished()
	if len(unfinished) == 0 {
		t.Fatal("expected unfinished tasks after stop (task id2)")
	}
	if len(unfinished) != 1 || unfinished[0].ID != id2 {
		t.Fatalf("expected task id2 to be unfinished, got: %v", unfinished)
	}

	mgr2 := worker.NewManager(store, 1)
	go mgr2.Run()

	time.Sleep(200 * time.Millisecond)
	mgr2.Stop()

	unfinished, _ = store.ListUnfinished()
	if len(unfinished) != 0 {
		t.Fatalf("expected all tasks done after restart, unfinished: %v", unfinished)
	}

	set1, _ := store.GetSet(id1)
	if set1.Results["http://link1.com"].State != models.StateAvailable {
		t.Errorf("link1 expected available, got %s", set1.Results["http://link1.com"].State)
	}
	if set1.Results["http://link2.com"].State != models.StateNotAvailable {
		t.Errorf("link2 expected not available, got %s", set1.Results["http://link2.com"].State)
	}

	set2, _ := store.GetSet(id2)
	if set2.Results["http://link3.com"].State != models.StateAvailable {
		t.Errorf("link3 expected available, got %s", set2.Results["http://link3.com"].State)
	}
}
