package outputs

import (
	"back/domain"
	"time"
)

type MensajeResponse struct {
	ID                  int64   `json:"id"`
	Telefono            string  `json:"telefono"`
	HoraSolicitada      string  `json:"hora_solicitada"`
	HoraDesactivacion   string  `json:"hora_desactivacion"`
	UsuarioAcargo       *string `json:"usuario_acargo"`
	HoraUsuarioAsignado *string `json:"hora_usuario_asignado"`
	Estado              string  `json:"estado"`
}

func ToMensajeResponse(m *domain.MensajePendiente) MensajeResponse {
	r := MensajeResponse{
		ID:                m.ID,
		Telefono:          m.Telefono,
		HoraSolicitada:    m.HoraSolicitada.Format(time.RFC3339),
		HoraDesactivacion: m.HoraDesactivacion.Format(time.RFC3339),
		Estado:            m.Estado,
	}
	if m.UsuarioAcargo != nil {
		r.UsuarioAcargo = m.UsuarioAcargo
	}
	if m.HoraUsuarioAsignado != nil {
		s := m.HoraUsuarioAsignado.Format(time.RFC3339)
		r.HoraUsuarioAsignado = &s
	}
	return r
}
