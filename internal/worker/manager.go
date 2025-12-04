package worker

import (
	"log"
	"sync"

	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/internal/util"
	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/models"
)

type StoreWorker interface {
	GetSet(int64) (*models.LinkSet, error)
	UpdateLinkResult(int64, string, models.LinkResult) error
	ListUnfinished() ([]*models.LinkSet, error)
}

type Manager struct {
	store   StoreWorker
	jobs    chan int64
	wg      sync.WaitGroup
	stop    chan struct{}
	workers int
}

func NewManager(st StoreWorker, workers int) *Manager {
	m := &Manager{
		store:   st,
		jobs:    make(chan int64, 1000),
		stop:    make(chan struct{}),
		workers: workers,
	}

	if unfinished, err := st.ListUnfinished(); err == nil {
		for _, s := range unfinished {
			m.Enqueue(s.ID)
		}
	}

	return m
}

func (m *Manager) Run() {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker()
	}
	m.wg.Wait()
}

func (m *Manager) Stop() {
	close(m.stop)
	close(m.jobs)
	m.wg.Wait()
}

func (m *Manager) Enqueue(id int64) {
	select {
	case m.jobs <- id:
	default:
		go func() { m.jobs <- id }()
	}
}

// worker проверяет ссылки для задач из очереди
func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		select {
		case <-m.stop:
			return
		case id, ok := <-m.jobs:
			if !ok {
				return
			}
			set, err := m.store.GetSet(id)
			if err != nil {
				log.Printf("load set %d: %v", id, err)
				continue
			}

			var wg sync.WaitGroup
			for _, url := range set.Links {
				res := set.Results[url]
				if res != nil && (res.State == models.StateAvailable || res.State == models.StateNotAvailable) {
					continue
				}

				wg.Go(func() {
					// помечаем ссылку как processing
					r := res
					if r == nil {
						r = &models.LinkResult{URL: url, State: models.StateProcessing}
					} else {
						r.State = models.StateProcessing
					}
					m.store.UpdateLinkResult(id, url, *r)

					ok, detail := util.CheckURL(url)
					result := models.LinkResult{
						URL:       url,
						CheckedAt: util.Now(),
						Detail:    detail,
					}
					if ok {
						result.State = models.StateAvailable
					} else {
						result.State = models.StateNotAvailable
					}

					if err := m.store.UpdateLinkResult(id, url, result); err != nil {
						log.Printf("update result %s: %v", url, err)
					}
				})
			}
			wg.Wait()
		}
	}
}
