package models

import "strconv"

type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"` // optional
	UserName     string `json:"username"`            // optional
	LanguageCode string `json:"language_code"`       // optional
	IsBot        bool   `json:"is_bot"`              // optional
}

func (t *TelegramUser) Name() string {
	switch {
	case t.UserName != "":
		return t.UserName
	case t.FirstName != "" && t.LastName != "":
		return t.FirstName + " " + t.LastName
	case t.FirstName != "":
		return t.FirstName
	case t.LastName != "":
		return t.LastName
	default:
		return strconv.FormatInt(t.ID, 10)
	}
}
