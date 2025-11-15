package logger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Env string `env:"LOGGER" env-required:"true"`
}

func New(cfg *Config) (*zap.Logger, error) {
	switch cfg.Env {
	case "dev":
		loggerConfig := zap.NewDevelopmentConfig()

		loggerConfig.DisableCaller = true
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		loggerConfig.EncoderConfig.LineEnding = "\n\n"
		loggerConfig.EncoderConfig.ConsoleSeparator = " | "
		loggerConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("\033[36m" + t.Format("15:04:05") + "\033[0m")
		}

		logger, err := loggerConfig.Build()
		if err != nil {
			return nil, err
		}

		return logger, nil

	case "prod":
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}

		return logger, nil

	default:
		return nil, fmt.Errorf("unknown environment: %s", cfg.Env)
	}
}

func MiddlewareLogger(logger *zap.Logger, cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := logger.With()
			start := time.Now()

			switch cfg.Env {
			case "dev":
				entry = logger.With(
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)

				entry.Info("new request")

			default:
				entry = logger.With(
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.Time("time", time.Now()),
				)

				entry.Info("new request")
			}
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				switch cfg.Env {
				case "dev":
					entry.Info(
						"request completed",
						zap.Int("status", ww.Status()),
					)

				default:
					entry.Info(
						"request completed",
						zap.Int("status", ww.Status()),
						zap.Duration("duration", time.Since(start)),
					)
				}
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
