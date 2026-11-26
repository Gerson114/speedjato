package entrada

// CadastroRequest são os dados de entrada do cadastro. As tags binding fazem
// o Gin validar automaticamente (campos obrigatórios, formato de email e o
// limite de 72 bytes do bcrypt).
type CadastroRequest struct {
	Username string `json:"username" binding:"required,max=200"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=15,max=72"`
}

// LoginRequest são os dados de entrada do login.
type LoginRequest struct {
	RememberMe bool   `json:"rememberMe"`
	Email      string `json:"email" binding:"required,email,max=254"`
	Password   string `json:"password" binding:"required"`
}
