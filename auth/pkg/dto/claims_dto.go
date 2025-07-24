package dto

type UserClaimsDTO struct {
	Username string   `json:"username"`
	Issuer   string   `json:"issuer"`
	Audience []string `json:"audience"`
	Subject  string   `json:"subject"`
	Rights   []string `json:"rights"`
}
