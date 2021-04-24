package models

type UserReq struct {
	Email string `json:"email" validate:"required,email"`
}

type BookmarkReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Link        *string `json:"link"`
}

type BookmarkResp struct {
	ID          uint    `json:"id"`
	Name        *string `json:"name,omitempty"`
	Link        *string `json:"link,omitempty"`
	Description *string `json:"description,omitempty"`
}
