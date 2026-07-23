package inputs

type MensajeInput struct {
	Telefono          string `json:"telefono"`
	HoraDesactivacion string `json:"hora_desactivacion"`
}

type MensajeUpdateInput struct {
	UsuarioAcargo *string `json:"usuario_acargo,omitempty"`
	Finalizar     *bool   `json:"finalizar,omitempty"`
}
