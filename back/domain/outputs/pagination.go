package outputs

type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Offset     int   `json:"offset"`
	Limit      int   `json:"limit"`
}
