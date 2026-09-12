package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	cachecli "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/cli"
	cachefilesystem "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/filesystemstore"
	cachehttp "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/httpapi"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/originhttp"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn/controlpanelclient"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn/filesystemstore"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn/memorystore"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn/natshandler"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/edgehealth"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/edgehealth/natspublisher"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/httpserver"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/natsbroker"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/observability"
	"github.com/gin-gonic/gin"
)

const appVersion = "v1.0.0"

func main() {
	cfg := config.Load()
	if len(os.Args) > 1 {
		if os.Args[1] != "purge-cache" {
			log.Fatalf("unknown command %q", os.Args[1])
		}
		if err := purgeCache(cfg.CacheDir, os.Args[2:]); err != nil {
			log.Fatalf("purge cache: %v", err)
		}
		return
	}
	gin.SetMode(cfg.GinMode)

	metrics := observability.NewMetrics()
	cacheStore, err := cachefilesystem.New(cfg.CacheDir, cfg.CleanerIntervalDuration, metrics)
	if err != nil {
		log.Fatalf("initialize cache store: %v", err)
	}
	defer cacheStore.Close()
	if err := cacheStore.Load(); err != nil {
		log.Printf("load cache metadata: %v", err)
	}

	controlPanelHTTPClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}
	defer controlPanelHTTPClient.CloseIdleConnections()
	originHTTPClient, err := originhttp.New(originhttp.Options{
		DialTimeout:           cfg.OriginDialTimeoutDuration,
		ResponseHeaderTimeout: cfg.OriginResponseHeaderTimeoutDuration,
		IdleConnTimeout:       cfg.OriginIdleConnTimeoutDuration,
		MaxIdleConns:          cfg.OriginMaxIdleConns,
		MaxIdleConnsPerHost:   cfg.OriginMaxIdlePerHost,
		MaxConcurrent:         cfg.OriginMaxConcurrent,
	}, cfg.OriginAllowedCIDRs)
	if err != nil {
		log.Fatalf("initialize origin HTTP client: %v", err)
	}
	defer originHTTPClient.CloseIdleConnections()
	cdnStore := memorystore.New()
	snapshotService := cdn.NewService(
		controlpanelclient.New(cfg.ControlPanelURL, cfg.EdgeServiceToken, controlPanelHTTPClient),
		cdnStore,
		filesystemstore.New(cfg.SnapshotFile),
		cfg.SyncIntervalDuration,
	)
	if err := snapshotService.LoadLocal(context.Background()); err != nil {
		log.Printf("load last-known-good CDN snapshot: %v", err)
	}
	initialCtx, initialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := snapshotService.Sync(initialCtx); err != nil {
		log.Printf("initial CDN synchronization failed; using last-known-good snapshot: %v", err)
	}
	initialCancel()

	cacheService := cache.NewService(
		cdnStore,
		cacheStore,
		originHTTPClient,
		metrics,
		cfg.CacheMaxObjectSizeBytes,
	)
	handler, err := cachehttp.NewWithTrustedProxies(cacheService, cfg.TrustedProxies)
	if err != nil {
		log.Fatalf("initialize proxy trust policy: %v", err)
	}
	httpLimits := httpserver.Limits{
		ReadHeaderTimeout:     cfg.HTTPReadHeaderTimeoutDuration,
		IdleTimeout:           cfg.HTTPIdleTimeoutDuration,
		MaxHeaderBytes:        cfg.HTTPMaxHeaderBytes,
		MaxConnections:        cfg.HTTPMaxConnections,
		MaxConcurrentRequests: cfg.HTTPMaxConcurrent,
	}
	publicServer := httpserver.NewPublic(cfg.AppCacheURL, handler, httpLimits)
	internalServer := httpserver.NewInternal(cfg.AppInternalURL, snapshotService, httpLimits)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go cacheStore.RunCleaner(ctx)
	go snapshotService.RunPeriodic(ctx, func(err error) {
		log.Printf("periodic CDN synchronization failed: %v", err)
	})

	broker, err := natsbroker.New(cfg.NATSURL)
	if err != nil {
		log.Fatalf("initialize NATS client: %v", err)
	}
	defer broker.Close()
	initiallyConnected := broker.Connected()
	if !initiallyConnected {
		log.Printf("NATS unavailable; retrying while periodic CDN sync continues")
	}
	if err := natshandler.New(broker, snapshotService).Register(ctx); err != nil {
		log.Printf("register CDN snapshot subscriber: %v", err)
	}
	go runHealthPublisher(ctx, broker, snapshotService, cfg.AppName, initiallyConnected)

	serverErrors := make(chan error, 2)
	go serve(publicServer, "public", serverErrors)
	go serve(internalServer, "internal", serverErrors)

	select {
	case <-ctx.Done():
	case err := <-serverErrors:
		log.Printf("edge server stopped unexpectedly: %v", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publicServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shut down public server: %v", err)
	}
	if err := internalServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shut down internal server: %v", err)
	}
}

func purgeCache(directory string, args []string) error {
	purger, err := cachefilesystem.NewPurger(directory)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return cachecli.NewPurgeCacheCommand(cache.NewPurgeService(purger)).Run(ctx, args)
}

func runHealthPublisher(ctx context.Context, broker *natsbroker.Broker, readiness edgehealth.Readiness, instance string, initiallyConnected bool) {
	if err := broker.WaitForConnection(ctx); err != nil {
		return
	}
	if !initiallyConnected {
		log.Printf("NATS connection recovered; messaging is active")
	}
	edgehealth.NewService(natspublisher.New(broker), readiness, "edge", instance, appVersion, 10*time.Second).Run(ctx)
}

func serve(server *http.Server, name string, errorsChannel chan<- error) {
	log.Printf("%s edge server listening on %s", name, server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errorsChannel <- fmt.Errorf("%s server: %w", name, err)
	}
}
