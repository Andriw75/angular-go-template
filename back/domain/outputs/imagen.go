package outputs

import "back/domain"

type ImagenResponse struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

func ToImagenResponses(imagenes []domain.Imagen) []ImagenResponse {
	res := make([]ImagenResponse, 0, len(imagenes))
	for _, i := range imagenes {
		res = append(res, ImagenResponse{ID: i.ID, URL: "/imagenes/" + i.FileName})
	}
	return res
}
