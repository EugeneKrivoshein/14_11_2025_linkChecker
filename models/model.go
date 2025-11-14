package models

import "time"

//тип состояния ссылки
type LinkState string

const (
	StateUnknown      LinkState = "unknown"    //ссылка еще не проверялась
	StateProcessing   LinkState = "processing" //в процессе проверки(worker)
	StateAvailable    LinkState = "available"
	StateNotAvailable LinkState = "not_available"
)

//результат проверки одной ссылки
type LinkResult struct {
	URL       string    `json:"url"`
	State     LinkState `json:"state"`
	CheckedAt time.Time `json:"checked_at,omitempty"`
	Detail    string    `json:"detail,omitempty"`
}

//набор ссылок отправленных одним запросом
type LinkSet struct {
	ID        int64                  `json:"id"`
	Links     []string               `json:"links"`
	Results   map[string]*LinkResult `json:"results"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Status    string                 `json:"status"`
}
