package outputs

import "back/domain"

type ImagenResponse struct {
	ID      int64  `json:"id"`
	URL     string `json:"url"`
	Orden   int    `json:"orden"`
	Publico bool   `json:"publico"`
}

func ToImagenResponses(imagenes []domain.Imagen, publico bool) []ImagenResponse {
	res := make([]ImagenResponse, 0, len(imagenes))
	for _, i := range imagenes {
		res = append(res, ImagenResponse{
			ID:      i.ID,
			URL:     "/imagenes/" + i.FileName,
			Orden:   i.Orden,
			Publico: publico,
		})
	}
	return res
}
