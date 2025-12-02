package validators

type CreateNoteRequest struct {
	Title     string `json:"title" binding:"required"`
	Content   string `json:"content" binding:"required"`
	IsPrivate bool   `json:"isPrivate"`
	TagIDs    []uint `json:"tag_ids"`
}

type UpdateNoteRequest struct {
	Title     string `json:"title" binding:"required"`
	Content   string `json:"content" binding:"required"`
	IsPrivate bool   `json:"isPrivate"`
	TagIDs    []uint `json:"tag_ids"`
}
