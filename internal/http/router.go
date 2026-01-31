package http

import (
	"log/slog"

	"avagostar-form-server/internal/config"
	"avagostar-form-server/internal/http/handlers"
	"avagostar-form-server/internal/http/middleware"
	"avagostar-form-server/internal/repo"
	"avagostar-form-server/internal/services"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Config      *config.Config
	UserRepo    *repo.UserRepo
	AuthService *services.AuthService
	TxService   *services.TransactionService
	Logger      *slog.Logger
	RateLimiter *middleware.RateLimiter
}

func NewRouter(deps Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(deps.Logger))
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(deps.Config.AllowedOrigins))

	authHandler := handlers.NewAuthHandler(deps.AuthService)
	meHandler := handlers.NewMeHandler(deps.UserRepo)
	txHandler := handlers.NewTransactionHandler(deps.TxService)
	userHandler := handlers.NewUserHandler(deps.AuthService)

	router.GET("/healthz", handlers.Health)

	api := router.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		authGroup.Use(deps.RateLimiter.Middleware())
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/forgot", authHandler.Forgot)
		authGroup.POST("/reset", authHandler.Reset)
	}

	protected := api.Group("")
	protected.Use(middleware.JWTAuth(middleware.AuthConfig{Secret: deps.Config.JWTSecret}))
	{
		protected.GET("/me", meHandler.GetMe)
		protected.POST("/users", userHandler.Create)
		protected.POST("/transactions", txHandler.Create)
		protected.GET("/transactions", txHandler.List)
		protected.GET("/transactions/summary", txHandler.Summary)
		protected.GET("/transactions/:id", txHandler.GetByID)
		protected.DELETE("/transactions/:id", txHandler.Delete)
	}

	return router
}
