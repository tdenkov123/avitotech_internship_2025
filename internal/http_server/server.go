package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tdenkov123/avitotech_internship_2025/internal/config"
	openapi "github.com/tdenkov123/avitotech_internship_2025/internal/http_server/api"
	"github.com/tdenkov123/avitotech_internship_2025/internal/http_server/handlers"
	"github.com/tdenkov123/avitotech_internship_2025/internal/http_server/middleware"
	"github.com/tdenkov123/avitotech_internship_2025/internal/service"
)

type Server struct {
	engine *gin.Engine
	server *http.Server
	logger *zap.Logger
	cfg    config.Config
}

func New(cfg config.Config, logger *zap.Logger, svc *service.Service) *Server {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logging(logger))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	apiHandler := handlers.NewAPIHandler(logger, svc)
	openapi.RegisterHandlers(engine, apiHandler)
	engine.POST("/team/deactivate", apiHandler.DeactivateTeamMembers)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: engine,
	}

	return &Server{
		engine: engine,
		server: srv,
		logger: logger,
		cfg:    cfg,
	}
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatal("http server failure", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return nil
}
