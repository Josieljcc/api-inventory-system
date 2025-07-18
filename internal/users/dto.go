package users

type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}
