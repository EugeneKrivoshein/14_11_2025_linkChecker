package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/models"
)

type FileStore struct {
	dir  string
	mu   sync.Mutex
	last int64 //последний id
}

func NewFileStore(dir string) (*FileStore, error) {
	fs := &FileStore{dir: dir}
	os.MkdirAll(filepath.Join(dir, "sets"), 0o755)
	//0o755 права доступа к папке. читать и открывать каталог всем
	//писать только владелец

	//если сервис перезагрузится загрузить последний id
	meta := filepath.Join(dir, "meta.json")
	if _, err := os.Stat(meta); err == nil {
		b, _ := os.ReadFile(meta)
		var m struct {
			Last int64 `json:"last"`
		}
		json.Unmarshal(b, &m)
		fs.last = m.Last
	}

	return fs, nil
}

func (f *FileStore) persistMeta() error {
	meta := filepath.Join(f.dir, "meta.json")
	tmp := meta + ".tmp" //защита от повреждения файла при падении

	b, _ := json.MarshalIndent(struct {
		Last int64 `json:"last"`
	}{f.last}, "", " ")

	//0o644 - владелец может читать и запись
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, meta) //атомарно заменяется на meta.json
}

func (f *FileStore) nextID() (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.last++
	if err := f.persistMeta(); err != nil {
		return 0, err
	}

	return f.last, nil
}

func (f *FileStore) setPath(id int64) string {
	return filepath.Join(f.dir, "sets", fmt.Sprintf("%d.json", id))
}

// безопасное созранение структуры в файл
func (f *FileStore) saveSet(s *models.LinkSet) error {
	p := f.setPath(s.ID)
	tmp := p + ".tmp"

	b, _ := json.MarshalIndent(s, "", " ")

	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, p)
}

func (f *FileStore) GetSet(id int64) (*models.LinkSet, error) {
	p := f.setPath(id)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var s models.LinkSet
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (f *FileStore) CreateSet(links []string) (int64, *models.LinkSet, error) {
	if len(links) == 0 {
		return 0, nil, fmt.Errorf("нет ссылок")
	}

	id, err := f.nextID()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get next ID: %v", err)
	}

	s := &models.LinkSet{
		ID:        id,
		Links:     links,
		Results:   make(map[string]*models.LinkResult),
		Status:    "processing",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := f.saveSet(s); err != nil {
		return 0, nil, fmt.Errorf("failed to save set: %v", err)
	}

	return id, s, nil
}

func (f *FileStore) UpdateLinkResult(id int64, url string, res models.LinkResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.GetSet(id)
	if err != nil {
		return err
	}

	if s.Results == nil {
		s.Results = map[string]*models.LinkResult{}
	}

	r := res
	s.Results[url] = &r //сохраняем результат по ключу url

	allDone := true //проверка все ли ссылки обработаны
	for _, rr := range s.Results {
		if rr.State != models.StateAvailable &&
			rr.State != models.StateNotAvailable {
			allDone = false
		}
	}

	if allDone {
		s.Status = "done"
	}

	s.UpdatedAt = time.Now()
	return f.saveSet(s)
}

// получить все незавершенные задачи
func (f *FileStore) ListUnfinished() ([]*models.LinkSet, error) {
	dir := filepath.Join(f.dir, "sets")
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []*models.LinkSet

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		b, _ := os.ReadFile(filepath.Join(dir, fi.Name()))
		var s models.LinkSet
		if json.Unmarshal(b, &s) == nil {
			if s.Status != "done" {
				out = append(out, &s) //возвращаются только задачи которые надо восстановить
			}
		}
	}

	return out, nil
}

func (f *FileStore) ListSets(ids []int64) ([]*models.LinkSet, error) {
	out := make([]*models.LinkSet, 0, len(ids))
	for _, id := range ids {
		s, err := f.GetSet(id)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
