package outputs

type PaginatedResponse[T any] struct {
	Data    []T  `json:"data"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"has_more"`
}
