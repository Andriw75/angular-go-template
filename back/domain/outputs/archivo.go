package outputs

import (
	"time"

	"back/domain"
)

type ArchivoResponse struct {
	ID          int64  `json:"id"`
	URL         string `json:"url"`
	Nombre      string `json:"nombre"`
	Tipo        string `json:"tipo"`
	Tamano      int64  `json:"tamano"`
	Visibilidad string `json:"visibilidad"`
	CreadoEn    string `json:"creado_en"`
}

func ToArchivoResponse(a *domain.Archivo) ArchivoResponse {
	return ArchivoResponse{
		ID:          a.ID,
		URL:         "/archivos/" + a.FileName,
		Nombre:      a.Nombre,
		Tipo:        a.Tipo,
		Tamano:      a.Tamano,
		Visibilidad: string(a.Visibilidad),
		CreadoEn:    a.CreadoEn.Format(time.RFC3339),
	}
}

func ToArchivoResponses(archivos []domain.Archivo) []ArchivoResponse {
	res := make([]ArchivoResponse, 0, len(archivos))
	for _, a := range archivos {
		res = append(res, ToArchivoResponse(&a))
	}
	return res
}
