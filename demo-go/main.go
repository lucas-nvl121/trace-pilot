package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

type Config struct {
	Addr           string
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string
	OTLPInsecure   bool
}

type Readiness struct {
	ready atomic.Bool
}

func NewConfig() (*Config, error) {
	insecure, err := envBool("OTEL_EXPORTER_OTLP_INSECURE", true)
	if err != nil {
		return nil, err
	}

	return &Config{
		Addr:           envString("HTTP_ADDR", ":8080"),
		ServiceName:    envString("SERVICE_NAME", "demo-go"),
		ServiceVersion: envString("SERVICE_VERSION", "dev"),
		OTLPEndpoint:   envString("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTLPInsecure:   insecure,
	}, nil
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseBool(value)
}

func NewLogger(cfg *Config) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(
		"service.name", cfg.ServiceName,
		"service.version", cfg.ServiceVersion,
	)
}

func NewTracerProvider(cfg *Config) (*sdktrace.TracerProvider, error) {
	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(context.Background(), exporterOptions...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	return provider, nil
}

func NewReadiness() *Readiness { return &Readiness{} }

func NewRouter(cfg *Config, logger *slog.Logger, r *Readiness) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware(cfg.ServiceName))
	router.Use(requestLogger(logger))
	router.GET("/", func(c *gin.Context) {
		_, span := otel.Tracer(cfg.ServiceName).Start(c.Request.Context(), "demo.root_handler")
		defer span.End()
		c.JSON(http.StatusOK, gin.H{"message": "Hello World!"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/readyz", func(c *gin.Context) {
		if r.ready.Load() {
			c.Status(http.StatusOK)
		} else {
			c.Status(http.StatusServiceUnavailable)
		}
	})
	return router
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			attrs = append(attrs,
				"trace_id", span.SpanContext().TraceID().String(),
				"span_id", span.SpanContext().SpanID().String(),
			)
		}
		logger.InfoContext(c.Request.Context(), "HTTP request", attrs...)
	}
}

func NewHTTPServer(cfg *Config, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func Run(lc fx.Lifecycle, server *http.Server, r *Readiness, cfg *Config, logger *slog.Logger, tracerProvider *sdktrace.TracerProvider) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.InfoContext(ctx, "Starting HTTP server", "addr", server.Addr)

			ln, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}
			// Set readiness to true
			r.ready.Store(true)

			go func() {
				err := server.Serve(ln)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP server stopped unexpectedly", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.InfoContext(ctx, "Stopping HTTP server")

			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			r.ready.Store(false)
			serverErr := server.Shutdown(ctx)
			tracerErr := tracerProvider.Shutdown(ctx)
			return errors.Join(serverErr, tracerErr)
		},
	})
}

func main() {
	fx.New(
		fx.Provide(
			NewConfig,
			NewLogger,
			NewTracerProvider,
			NewReadiness,
			NewRouter,
			NewHTTPServer,
		),
		fx.Invoke(Run),
	).Run()
}
