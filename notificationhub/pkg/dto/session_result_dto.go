package dto

type SessionFinishedEvent struct {
	UserID     string              `json:"user_id"`
	Topics     []string            `json:"topics"`
	Questions  map[string][]string `json:"questions"`
	UserAnswer map[string][]string `json:"user_answer"`
	IsExpire   bool                `json:"is_expire"`
	IsSuccess  bool                `json:"is_success"`
	Resume     string              `json:"resume"`
}
