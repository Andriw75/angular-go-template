package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"back/domain/outputs"
	"back/services"
	"back/store"
)

type DocumentoHandler struct {
	deps *Dependencies
	svc  *services.DocumentoService
}

func NewDocumentoHandler(deps *Dependencies) *DocumentoHandler {
	reports := services.NewReporteService(deps.BusStore, deps.VentaStore)
	return &DocumentoHandler{
		deps: deps,
		svc:  services.NewDocumentoService(deps.DocumentoStore, deps.VentaStore, reports, deps.Config),
	}
}

func (h *DocumentoHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 20
	}
	filters := store.DocumentoFilters{
		Q:      q.Get("q"),
		Tipo:   q.Get("tipo"),
		Serie:  q.Get("serie"),
		Estado: q.Get("estado"),
	}
	docs, hasMore, err := h.deps.DocumentoStore.List(offset, limit, filters)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al listar documentos")
		return
	}
	res := make([]outputs.DocumentoResponse, 0, len(docs))
	for i := range docs {
		res = append(res, outputs.ToDocumentoResponse(&docs[i]))
	}
	writeJSON(w, http.StatusOK, outputs.PaginatedResponse[outputs.DocumentoResponse]{
		Data:    res,
		Offset:  offset,
		Limit:   limit,
		HasMore: hasMore,
	})
}

func (h *DocumentoHandler) Count(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filters := store.DocumentoFilters{
		Q:      q.Get("q"),
		Tipo:   q.Get("tipo"),
		Serie:  q.Get("serie"),
		Estado: q.Get("estado"),
	}
	total, err := h.deps.DocumentoStore.Count(filters)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al contar documentos")
		return
	}
	writeJSON(w, http.StatusOK, total)
}

func (h *DocumentoHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	doc, err := h.deps.DocumentoStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "documento not found")
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToDocumentoResponse(doc))
}

func (h *DocumentoHandler) File(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	doc, err := h.deps.DocumentoStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "documento not found")
		return
	}
	formato := r.URL.Query().Get("formato")
	if formato == "" {
		formato = "pdf"
	}
	var selected *struct {
		FileName string
		Mime     string
		Version  int
	}
	for i := range doc.Archivos {
		a := &doc.Archivos[i]
		if a.Formato != formato {
			continue
		}
		if selected == nil || a.Version > selected.Version {
			selected = &struct {
				FileName string
				Mime     string
				Version  int
			}{a.FileName, a.Mime, a.Version}
		}
	}
	if selected == nil {
		writeJSONError(w, http.StatusNotFound, "archivo not found")
		return
	}
	path := filepath.Join(h.deps.Config.DocumentosDir(), selected.FileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		writeJSONError(w, http.StatusNotFound, "archivo no existe")
		return
	}
	w.Header().Set("Content-Type", selected.Mime)
	w.Header().Set("Content-Disposition", "attachment; filename="+selected.FileName)
	http.ServeFile(w, r, path)
}

func (h *DocumentoHandler) Annul(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Motivo string `json:"motivo"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	claims := GetClaims(r.Context())
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}
	doc, err := h.deps.DocumentoStore.Annul(id, body.Motivo, userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToDocumentoResponse(&doc))
}

func (h *DocumentoHandler) GenerateBuses(w http.ResponseWriter, r *http.Request) {
	var p services.BusesDocumentParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	claims := GetClaims(r.Context())
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}

	job := h.svc.Jobs().Create("documento_buses", func(job *services.Job) {
		doc, exists, err := h.svc.GenerateBuses(context.Background(), p, userID, func(progress int) {
			job.Progreso = progress
			job.Actualizado = time.Now()
		})
		if err != nil {
			h.svc.Jobs().Update(job, "error", 100, err.Error(), nil, false)
			return
		}
		h.svc.Jobs().Update(job, "listo", 100, "", &doc, exists)
	})
	h.respondJobOrWait(w, job)
}

func (h *DocumentoHandler) GenerateVenta(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p services.VentaDocumentParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p.VentaID = id
	claims := GetClaims(r.Context())
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}

	job := h.svc.Jobs().Create("documento_venta", func(job *services.Job) {
		doc, exists, err := h.svc.GenerateVenta(context.Background(), p, userID, func(progress int) {
			job.Progreso = progress
			job.Actualizado = time.Now()
		})
		if err != nil {
			h.svc.Jobs().Update(job, "error", 100, err.Error(), nil, false)
			return
		}
		h.svc.Jobs().Update(job, "listo", 100, "", &doc, exists)
	})
	h.respondJobOrWait(w, job)
}

func (h *DocumentoHandler) respondJobOrWait(w http.ResponseWriter, job *services.Job) {
	select {
	case <-job.Done:
		if job.Estado == "error" {
			writeJSONError(w, http.StatusInternalServerError, job.Error)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"job_id":    job.ID,
			"exists":    job.Exists,
			"documento": outputs.ToDocumentoResponse(job.Documento),
		})
	case <-time.After(h.svc.SyncTimeout()):
		writeJSON(w, http.StatusAccepted, map[string]any{"job_id": job.ID, "estado": job.Estado, "progreso": job.Progreso})
	}
}

func (h *DocumentoHandler) Job(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")
	job, ok := h.svc.Jobs().Get(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "job not found")
		return
	}
	resp := map[string]any{
		"id": job.ID, "estado": job.Estado, "progreso": job.Progreso, "error": job.Error,
		"exists": job.Exists,
	}
	if job.Documento != nil {
		resp["documento"] = outputs.ToDocumentoResponse(job.Documento)
	}
	writeJSON(w, http.StatusOK, resp)
}
