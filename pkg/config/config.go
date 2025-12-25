package config

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Veeam           Veeam  `yaml:"veeam"`
	Influx          Influx `yaml:"influx"`
	LogLevel        string `yaml:"log_level"`
	IntervalSeconds int    `yaml:"interval_seconds"`
}

type Veeam struct {
	Host                string              `yaml:"host"`
	XApiVersion         string              `yaml:"x_api_version"`
	TrustSelfSignedCert bool                `yaml:"trust_self_signed_cert"`
	Username            string              `json:"username"`
	Password            string              `json:"password"`
	ExcludedJobTypes    map[string]struct{} `yaml:"excluded_job_types"`
}

type Influx struct {
	Host   string `yaml:"host"`
	Token  string `yaml:"token"`
	Org    string `yaml:"org"`
	Bucket string `yaml:"bucket"`
}

var ErrConfigFileExported = errors.New("config file example created")

func NewConfig() (Config, error) {
	confFile := flag.String("config", "config.yaml", "Path to config file")
	exportConfig := flag.Bool("export", false, "Export config file with default values")
	flag.Parse()

	// default config
	config := Config{
		Veeam: Veeam{
			Host:                "https://veeam.server:9419",
			XApiVersion:         "1.2-rev0",
			TrustSelfSignedCert: false,
			Username:            "<veeam-admin or VEEAM_ADMIN_USERNAME>",
			Password:            "<veeam-admin-password or VEEAM_ADMIN_PASSWORD>",
			ExcludedJobTypes: map[string]struct{}{
				"MalwareDetection":           {},
				"SecurityComplianceAnalyzer": {},
			},
		},
		Influx: Influx{
			Host:   "http://influxdb:8086",
			Token:  "<influxdb-token or INFLUXDB_TOKEN>",
			Org:    "<influxdb-org-name or INFLUXDB_ORG_NAME>",
			Bucket: "<influxdb-bucket-name>",
		},
		LogLevel:        "INFO",
		IntervalSeconds: 3600,
	}

	// export config.yaml example
	if *exportConfig {
		f, err := os.Create("config.yaml")
		if err != nil {
			return Config{}, fmt.Errorf("error creating config file: %v", err)
		}

		if err = yaml.NewEncoder(f).Encode(config); err != nil {
			return Config{}, fmt.Errorf("error encoding config file: %v", err)
		}

		slog.Info("Config file example created", "file", f.Name())

		return Config{}, ErrConfigFileExported
	}

	// load config.yaml
	f, err := os.Open(*confFile)
	if err != nil {
		flag.PrintDefaults()
		return Config{}, fmt.Errorf("error opening config file: %v", err)
	}

	if err = yaml.NewDecoder(f).Decode(&config); err != nil {
		flag.PrintDefaults()
		return Config{}, fmt.Errorf("error parsing config file: %v", err)
	}

	// load env vars
	veeamUser := os.Getenv("VEEAM_ADMIN_USERNAME")
	if veeamUser != "" {
		config.Veeam.Username = veeamUser
	}

	veeamPassword := os.Getenv("VEEAM_ADMIN_PASSWORD")
	if veeamPassword != "" {
		config.Veeam.Password = veeamPassword
	}

	influxToken := os.Getenv("INFLUXDB_TOKEN")
	if influxToken != "" {
		config.Influx.Token = influxToken
	}

	influxOrg := os.Getenv("INFLUXDB_ORG_NAME")
	if influxOrg != "" {
		config.Influx.Org = influxOrg
	}

	return config, nil
}
