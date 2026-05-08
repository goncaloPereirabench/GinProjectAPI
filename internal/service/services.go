package service

type Services struct {
	Auth    *AuthService
	Product *ProductService
	Cart    *CartService
	JWT     *JWTManager
}
