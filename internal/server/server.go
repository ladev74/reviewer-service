package server

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"reviewer-service/internal/api/handler"
	"reviewer-service/internal/logger"
	"reviewer-service/internal/repository"
)

type Config struct {
	Host    string        `env:"HTTP_HOST" env-required:"true"`
	Port    int           `env:"HTTP_PORT" env-required:"true"`
	Timeout time.Duration `env:"HTTP_TIMEOUT" env-required:"true"`
}

func NewRouter(repo repository.Repository, log *zap.Logger, cfgLogger *logger.Config, srvTimeout time.Duration) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(logger.MiddlewareLogger(log, cfgLogger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/team/add", handler.AddTeam(repo, srvTimeout, log))
	router.Get("/team/get", handler.GetTeam(repo, srvTimeout, log))
	router.Post("/users/setIsActive", handler.SetIsActive(repo, srvTimeout, log))

	return router
}
