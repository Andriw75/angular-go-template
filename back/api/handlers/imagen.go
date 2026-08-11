package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"back/domain/outputs"
)

var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

const maxUploadSize = 64 << 20

func uploadImagesForEntity(deps *Dependencies, tipo string, entidadID int64, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, "error al parsear formulario")
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		writeJSONError(w, http.StatusBadRequest, "images requerido")
		return
	}

	prefix := fmt.Sprintf("%s_%d", tipo, entidadID)
	filenames, err := deps.Storage.SaveFiles(files, prefix, imageExts)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var created []int64
	rollback := func() {
		for _, cid := range created {
			if _, ferr := deps.ImagenStore.FindByID(cid); ferr == nil {
				_ = deps.ImagenStore.Delete(cid)
			}
		}
		deps.Storage.RemoveImages(filenames...)
	}

	for _, fn := range filenames {
		img, err := deps.ImagenStore.Add(tipo, entidadID, fn)
		if err != nil {
			rollback()
			writeJSONError(w, http.StatusInternalServerError, "error al registrar imagen")
			return
		}
		created = append(created, img.ID)
	}

	writeJSON(w, http.StatusCreated, outputs.ToImagenResponses(deps.ImagenStore.ListByEntidad(tipo, entidadID)))
}

func deleteImageForEntity(deps *Dependencies, tipo string, entidadID, imagenID int64, w http.ResponseWriter) {
	img, err := deps.ImagenStore.FindByID(imagenID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "imagen not found")
		return
	}
	if img.Tipo != tipo || img.EntidadID != entidadID {
		writeJSONError(w, http.StatusNotFound, "imagen not found")
		return
	}

	// Primero la baja en el store; solo tras éxito se borra el archivo.
	if err := deps.ImagenStore.Delete(imagenID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete imagen")
		return
	}
	deps.Storage.RemoveImages(img.FileName)

	writeJSON(w, http.StatusOK, map[string]string{"message": "imagen deleted"})
}

func parseImagenParams(r *http.Request) (int64, int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	imagenID, err := strconv.ParseInt(chi.URLParam(r, "imagenID"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return id, imagenID, true
}

func reorderImagesForEntity(deps *Dependencies, tipo string, entidadID int64, w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := deps.ImagenStore.Reorder(tipo, entidadID, body.IDs); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToImagenResponses(deps.ImagenStore.ListByEntidad(tipo, entidadID)))
}
