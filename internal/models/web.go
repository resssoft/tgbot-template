package models

type Response struct {
	Error  string `json:"error"`
	Status bool   `json:"status"`
	Code   int    `json:"code"`
}
