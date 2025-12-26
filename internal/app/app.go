package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZeljkoBenovic/govein/pkg/config"
	"github.com/ZeljkoBenovic/govein/pkg/influx"
	"github.com/ZeljkoBenovic/govein/pkg/veeam"
)

type App struct {
	influx         *influx.Influx
	veeam          *veeam.Veeam
	conf           config.Config
	ctx            context.Context
	log            *slog.Logger
	healthCheckErr chan error
}

func New() (*App, error) {
	ctx := context.Background()
	conf, err := config.NewConfig()
	if err != nil {
		if errors.Is(err, config.ErrConfigFileExported) {
			os.Exit(0)
		}

		return nil, fmt.Errorf("could not create config: %v", err)
	}

	var logLevel slog.Level
	if err = logLevel.UnmarshalText([]byte(conf.LogLevel)); err != nil {
		return nil, fmt.Errorf("could not parse log level: %v", err)
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	v, err := veeam.NewVeeam(ctx, conf, log)
	if err != nil {
		return nil, fmt.Errorf("could not create veeam client: %v", err)
	}

	if err = v.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to veeam server: %v", err)
	}

	i, err := influx.NewInflux(ctx, conf, log)
	if err != nil {
		return nil, fmt.Errorf("could not create influx client: %v", err)
	}

	return &App{
		ctx:            ctx,
		log:            log,
		conf:           conf,
		veeam:          v,
		influx:         i,
		healthCheckErr: make(chan error),
	}, nil
}

func (a *App) Run() error {
	a.log.Info("Veeam metrics collector started")

	tick := time.Tick(time.Duration(a.conf.IntervalSeconds) * time.Second)
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		a.log.Info("Shutdown signal received")
		a.influx.FlushAndClose()
		os.Exit(0)
	}()

	go a.runHealthcheckHTTPEndpoint()

	a.log.Info("Gathering data on time interval", "seconds", a.conf.IntervalSeconds)

	for {
		if err := a.veeam.GetSessions(); err != nil {
			return err
		}

		if err := a.veeam.GetManagedServers(); err != nil {
			return err
		}

		if err := a.veeam.GetRepositories(); err != nil {
			return err
		}

		if err := a.veeam.GetProxies(); err != nil {
			return err
		}

		if err := a.veeam.GetBackupObjects(); err != nil {
			return err
		}

		a.log.Info("Storing data...")
		if err := a.influx.SetVeeamServerInfo(a.veeam.ServerInfo); err != nil {
			return err
		}

		if err := a.influx.SetVeeamSessions(a.veeam.Sessions); err != nil {
			return err
		}

		if err := a.influx.SetManagedServers(a.veeam.ManagedSevers); err != nil {
			return err
		}

		if err := a.influx.SetRepositories(*a.veeam); err != nil {
			return err
		}

		if err := a.influx.SetProxies(a.veeam.Proxies); err != nil {
			return err
		}

		if err := a.influx.SetBackupObjects(a.veeam.BackupObjects); err != nil {
			return err
		}

		if err := a.influx.FlushAndClose(); err != nil {
			return err
		}

		a.log.Info("Veeam metrics collection successfully completed")
		select {
		case <-a.ctx.Done():
			return nil
		case <-tick:
			continue
		case err := <-a.healthCheckErr:
			return err
		}
	}
}

func (a *App) runHealthcheckHTTPEndpoint() {
	http.HandleFunc(a.conf.HealthCheckEndpoint, func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		a.log.Info("Running health check probe")

		if err := a.influx.Ping(); err != nil {
			resp := map[string]string{"status": "error", "component": "influxdb", "error": err.Error()}
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			a.healthCheckErr <- err
			return
		}

		if err := a.veeam.Ping(); err != nil {
			resp := map[string]string{"status": "error", "component": "veeam", "error": err.Error()}
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			a.healthCheckErr <- err
			return
		}

		resp := map[string]string{"status": "ok"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)

		end := time.Now()
		a.log.Info("Health check endpoint done", "done_at", end.Format(time.RFC3339), "duration", fmt.Sprintf("%dms", end.Sub(start).Milliseconds()))
	})

	a.log.Info("Health check endpoint started", "port", a.conf.HealthCheckPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", a.conf.HealthCheckPort), nil); err != nil {
		a.log.Error("Failed to start healthcheck endpoint", "error", err)
		os.Exit(1)
	}
}
