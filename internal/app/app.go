package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZeljkoBenovic/govein/pkg/config"
	"github.com/ZeljkoBenovic/govein/pkg/influx"
	"github.com/ZeljkoBenovic/govein/pkg/veeam"
)

type App struct {
	influx *influx.Influx
	veeam  *veeam.Veeam
	conf   config.Config
	ctx    context.Context
	log    *slog.Logger
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
		ctx:    ctx,
		log:    log,
		conf:   conf,
		veeam:  v,
		influx: i,
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
		<-tick
	}
}
