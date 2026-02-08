package note

import (
	"note/internal/svc"
)

type NoteHandler struct {
	svc *svc.ServiceContext
}

func NewNoteHandler(svc *svc.ServiceContext) *NoteHandler {
	return &NoteHandler{svc: svc}
}
