package dto

type SigninRequestDTO struct {
	Login    string `json:"login" example:"user@test.ru"`
	Password string `json:"password" example:"password123"`
}

type SigninResponseDTO struct {
	Token string `json:"token"`
}

type DynamicRegistrationDTO struct {
	UserID   string `json:"id" example:"user@test.ru"`
	Provider string `json:"provider" example:"email"`
}
