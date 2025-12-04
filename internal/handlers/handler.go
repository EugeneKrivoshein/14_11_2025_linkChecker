package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/internal/pdfgen"
	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/internal/util"
	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/models"
)

type LinkCreator interface {
	CreateSet([]string) (int64, *models.LinkSet, error)
	UpdateLinkResult(int64, string, models.LinkResult) error
	ListSets([]int64) ([]*models.LinkSet, error)
}

type Handler struct {
	store LinkCreator
	mgr   interface{ Enqueue(int64) } //worker ставит id набора ссылок в очередь
}

func NewHandler(s LinkCreator, mgr interface{ Enqueue(int64) }) *Handler {
	return &Handler{store: s, mgr: mgr}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var body map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		h.respondError(w, http.StatusBadRequest, "bad json")
		return
	}

	if raw, ok := body["links"]; ok { //ссылки
		h.handleLinks(w, raw)
		return
	}

	if raw, ok := body["links_list"]; ok { //список id для получения pdf
		h.handlePDF(w, raw)
		return
	}

	h.respondError(w, http.StatusBadRequest, "bad payload")
}

func (h *Handler) handleLinks(w http.ResponseWriter, raw json.RawMessage) {
	var links []string
	if err := json.Unmarshal(raw, &links); err != nil {
		h.respondError(w, http.StatusBadRequest, "bad links format")
		return
	}
	if len(links) == 0 {
		h.respondError(w, http.StatusBadRequest, "нет ссылок")
		return
	}

	id, _, err := h.store.CreateSet(links) //сохранение ссылок в filestore
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	out := make(map[string]string)

	for _, url := range links {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			ok, detail := util.CheckURL(u) // каждая ссылка проверяется в отдельной горутине
			res := models.LinkResult{
				URL:       u,
				CheckedAt: time.Now(),
				Detail:    detail,
			}
			if ok {
				res.State = models.StateAvailable
			} else {
				res.State = models.StateNotAvailable
			}

			if err := h.store.UpdateLinkResult(id, u, res); err != nil {
				log.Printf("update result: %v", err)
			}

			mu.Lock()
			if ok {
				out[u] = "available"
			} else {
				out[u] = "not available"
			}
			mu.Unlock()
		}(url)
	}

	wg.Wait()

	resp := map[string]any{
		"links":     out,
		"links_num": id,
	}
	h.respondJSON(w, http.StatusOK, resp)

	h.mgr.Enqueue(id) //id набора ставится в очередь на случай перезапуска сервиса
}

func (h *Handler) handlePDF(w http.ResponseWriter, raw json.RawMessage) {
	var ids []int64
	json.Unmarshal(raw, &ids)

	sets, err := h.store.ListSets(ids) //получить данные по id
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	buf, err := pdfgen.GeneratePDF(sets) //сгенерировать pdf
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.Write(buf)
}

func (h *Handler) respondJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) respondError(w http.ResponseWriter, code int, msg string) {
	log.Printf("error %d: %s", code, msg)
	h.respondJSON(w, code, map[string]string{"error": msg})
}
