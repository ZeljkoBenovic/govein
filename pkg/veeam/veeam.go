package veeam

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeljkoBenovic/govein/pkg/config"
	"github.com/deepmap/oapi-codegen/v2/pkg/securityprovider"
	"github.com/google/uuid"
	"github.com/veeamhub/veeam-vbr-sdk-go/v2/pkg/client"
)

type Veeam struct {
	ctx  context.Context
	conf config.Config
	log  *slog.Logger
	cl   *client.ClientWithResponses

	ServerInfo      ServerInfo
	Sessions        Sessions
	ManagedSevers   ManagedSevers
	Repositories    []SingleRepository
	AllRepositories AllRepositories
	Proxies         Proxies
	BackupObjects   BackupObjects
}

type ServerInfo struct {
	VbrId                  string `json:"vbrId"`
	Name                   string `json:"name"`
	BuildVersion           string `json:"buildVersion"`
	Patches                []any  `json:"patches"`
	DatabaseVendor         string `json:"databaseVendor"`
	SQLServerEdition       string `json:"sqlServerEdition"`
	SQLServerVersion       string `json:"sqlServerVersion"`
	DatabaseSchemaVersion  string `json:"databaseSchemaVersion"`
	DatabaseContentVersion string `json:"databaseContentVersion"`
}

type Sessions struct {
	Data []struct {
		SessionType     string    `json:"sessionType"`
		State           string    `json:"state"`
		PlatformName    string    `json:"platformName"`
		ID              string    `json:"id"`
		Name            string    `json:"name"`
		JobID           string    `json:"jobId"`
		CreationTime    time.Time `json:"creationTime"`
		EndTime         time.Time `json:"endTime"`
		ProgressPercent int       `json:"progressPercent"`
		Result          struct {
			Result     string `json:"result"`
			Message    string `json:"message"`
			IsCanceled bool   `json:"isCanceled"`
		} `json:"result"`
		ResourceID        string      `json:"resourceId"`
		ResourceReference string      `json:"resourceReference"`
		ParentSessionID   interface{} `json:"parentSessionId"`
		Usn               int         `json:"usn"`
		PlatformID        string      `json:"platformId"`
	} `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type ManagedSevers struct {
	Data       []ManagedSeversData `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

type ManagedSeversData struct {
	ViHostType      *string          `json:"viHostType,omitempty"`
	CredentialsID   string           `json:"credentialsId"`
	Port            *int64           `json:"port,omitempty"`
	Type            string           `json:"type"`
	Status          string           `json:"status"`
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	NetworkSettings *NetworkSettings `json:"networkSettings,omitempty"`
}

type NetworkSettings struct {
	Components     []Component `json:"components"`
	PortRangeStart int64       `json:"portRangeStart"`
	PortRangeEnd   int64       `json:"portRangeEnd"`
	ServerSide     bool        `json:"serverSide"`
}

type Component struct {
	ComponentName string `json:"componentName"`
	Port          int64  `json:"port"`
}

type Pagination struct {
	Total int64 `json:"total"`
	Count int64 `json:"count"`
	Skip  int64 `json:"skip"`
	Limit int64 `json:"limit"`
}

type SingleRepository struct {
	Data       []SingleRepositoryData `json:"data"`
	Pagination Pagination             `json:"pagination"`
}

type SingleRepositoryData struct {
	Type        string  `json:"type"`
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	HostID      string  `json:"hostId"`
	HostName    string  `json:"hostName"`
	Path        string  `json:"path"`
	CapacityGB  float64 `json:"capacityGB"`
	FreeGB      float64 `json:"freeGB"`
	UsedSpaceGB float64 `json:"usedSpaceGB"`
	IsOnline    bool    `json:"isOnline"`
}

type AllRepositories struct {
	Data       []RepositoriesData `json:"data"`
	Pagination Pagination         `json:"pagination"`
}

type RepositoriesData struct {
	HostID      *string     `json:"hostId,omitempty"`
	Repository  Repository  `json:"repository"`
	MountServer MountServer `json:"mountServer"`
	Type        string      `json:"type"`
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	UniqueID    string      `json:"uniqueId"`
	Share       *Share      `json:"share,omitempty"`
}

type MountServer struct {
	MountServerID         string                `json:"mountServerId"`
	WriteCacheFolder      string                `json:"writeCacheFolder"`
	VPowerNFSEnabled      bool                  `json:"vPowerNFSEnabled"`
	VPowerNFSPortSettings VPowerNFSPortSettings `json:"vPowerNFSPortSettings"`
}

type VPowerNFSPortSettings struct {
	MountPort     int64 `json:"mountPort"`
	VPowerNFSPort int64 `json:"vPowerNFSPort"`
}

type Repository struct {
	Path                  *string          `json:"path,omitempty"`
	TaskLimitEnabled      bool             `json:"taskLimitEnabled"`
	MaxTaskCount          int64            `json:"maxTaskCount"`
	ReadWriteLimitEnabled bool             `json:"readWriteLimitEnabled"`
	ReadWriteRate         int64            `json:"readWriteRate"`
	AdvancedSettings      AdvancedSettings `json:"advancedSettings"`
}

type AdvancedSettings struct {
	RotatedDriveCleanupMode string `json:"RotatedDriveCleanupMode"`
	AlignDataBlocks         bool   `json:"alignDataBlocks"`
	DecompressBeforeStoring bool   `json:"decompressBeforeStoring"`
	RotatedDrives           bool   `json:"rotatedDrives"`
	PerVMBackup             bool   `json:"perVmBackup"`
}

type Share struct {
	SharePath     string        `json:"sharePath"`
	CredentialsID string        `json:"credentialsId"`
	GatewayServer GatewayServer `json:"gatewayServer"`
}

type Proxies struct {
	Data       []ProxiesData `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

type ProxiesData struct {
	Server      Server `json:"server"`
	Type        string `json:"type"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Server struct {
	TransportMode         string              `json:"transportMode"`
	HostID                string              `json:"hostId"`
	FailoverToNetwork     bool                `json:"failoverToNetwork"`
	HostToProxyEncryption bool                `json:"hostToProxyEncryption"`
	ConnectedDatastores   ConnectedDatastores `json:"connectedDatastores"`
	MaxTaskCount          int64               `json:"maxTaskCount"`
}

type ConnectedDatastores struct {
	AutoSelectEnabled bool          `json:"autoSelectEnabled"`
	Datastores        []interface{} `json:"datastores"`
}

type GatewayServer struct {
	AutoSelectEnabled bool          `json:"autoSelectEnabled"`
	GatewayServerIDS  []interface{} `json:"gatewayServerIds"`
}

type BackupObjects struct {
	Data       []BackupObjectsData `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

type BackupObjectsData struct {
	ViType             ViType       `json:"viType"`
	ObjectID           string       `json:"objectId"`
	Path               string       `json:"path"`
	PlatformName       PlatformName `json:"platformName"`
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	Type               Type         `json:"type"`
	PlatformID         string       `json:"platformId"`
	RestorePointsCount int64        `json:"restorePointsCount"`
}

type PlatformName string

const (
	VMware PlatformName = "VMware"
)

type Type string

const (
	VM Type = "VM"
)

type ViType string

const (
	VirtualMachine ViType = "VirtualMachine"
)

func NewVeeam(ctx context.Context, conf config.Config, log *slog.Logger) (*Veeam, error) {
	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.Veeam.TrustSelfSignedCert,
			},
		},
	}

	cl, err := client.NewClientWithResponses(conf.Veeam.Host, client.WithHTTPClient(tlsClient))
	if err != nil {
		return nil, err
	}

	rl, err := cl.CreateTokenWithFormdataBodyWithResponse(context.Background(), &client.CreateTokenParams{
		XApiVersion: conf.Veeam.XApiVersion,
	}, client.CreateTokenFormdataRequestBody{
		GrantType: "password",
		Username:  &conf.Veeam.Username,
		Password:  &conf.Veeam.Password,
	})
	if err != nil {
		return nil, err
	}

	if rl.JSON200 == nil {
		log.Error("Error creating Veeam Token", "auth_response", string(rl.Body))
		return nil, errors.New("error creating Veeam Token")
	}

	bearerTokenProvider, bearerTokenProviderErr := securityprovider.NewSecurityProviderBearerToken(rl.JSON200.AccessToken)
	if bearerTokenProviderErr != nil {
		panic(bearerTokenProviderErr)
	}

	authcl, err := client.NewClientWithResponses(
		conf.Veeam.Host,
		client.WithRequestEditorFn(bearerTokenProvider.Intercept),
		client.WithHTTPClient(tlsClient),
	)
	if err != nil {
		return nil, err
	}

	return &Veeam{
		ctx:          ctx,
		conf:         conf,
		cl:           authcl,
		log:          log.WithGroup("veeam"),
		ServerInfo:   ServerInfo{},
		Repositories: make([]SingleRepository, 0),
	}, nil
}

func (v *Veeam) Ping() error {
	v.log.Info("Collecting veeam server info")

	rsi, err := v.cl.GetServerInfoWithResponse(v.ctx, &client.GetServerInfoParams{XApiVersion: v.conf.Veeam.XApiVersion})
	if err != nil {
		return err
	}

	if rsi.StatusCode() != http.StatusOK {
		return fmt.Errorf("veeam server status check failed: %s", rsi.Status())
	}

	v.log.Info("Veeam server status check", "status", rsi.Status())

	if err = json.NewDecoder(bytes.NewBuffer(rsi.Body)).Decode(&v.ServerInfo); err != nil {
		return fmt.Errorf("could not parse veeam server response: %v", err)
	}

	v.log.Info("Veeam server information", "name", v.ServerInfo.Name, "buildVersion", v.ServerInfo.BuildVersion)

	return nil
}

func (v *Veeam) GetSessions() error {
	v.log.Info("Collecting sessions information")

	resp, err := v.cl.GetAllSessionsWithResponse(v.ctx, &client.GetAllSessionsParams{
		XApiVersion: v.conf.Veeam.XApiVersion,
	})
	if err != nil {
		return fmt.Errorf("could not get sessions: %v", err)
	}

	var ses Sessions

	if err = json.NewDecoder(bytes.NewBuffer(resp.Body)).Decode(&ses); err != nil {
		return fmt.Errorf("could not parse sessions: %v", err)
	}

	v.Sessions = ses
	return nil
}

func (v *Veeam) GetManagedServers() error {
	v.log.Info("Collecting managed servers information")

	resp, err := v.cl.GetAllManagedServersWithResponse(v.ctx, &client.GetAllManagedServersParams{
		XApiVersion: v.conf.Veeam.XApiVersion,
	})
	if err != nil {
		return fmt.Errorf("could not get managed servers: %v", err)
	}

	var servers ManagedSevers

	if err = json.NewDecoder(bytes.NewBuffer(resp.Body)).Decode(&servers); err != nil {
		return fmt.Errorf("could not parse managed servers: %v", err)
	}

	v.ManagedSevers = servers

	return nil
}

func (v *Veeam) GetRepositories() error {
	v.log.Info("Collecting repositories information")

	resp, err := v.cl.GetAllRepositoriesWithResponse(v.ctx, &client.GetAllRepositoriesParams{
		XApiVersion: v.conf.Veeam.XApiVersion,
	})
	if err != nil {
		return fmt.Errorf("could not get repositories: %v", err)
	}

	var repos AllRepositories
	if err = json.NewDecoder(bytes.NewBuffer(resp.Body)).Decode(&repos); err != nil {
		return fmt.Errorf("could not parse repositories: %v", err)
	}

	for _, r := range repos.Data {
		uid, err := uuid.Parse(r.ID)
		if err != nil {
			return fmt.Errorf("could not parse repositories uuid: %v", err)
		}

		rsr, err := v.cl.GetAllRepositoriesStatesWithResponse(v.ctx, &client.GetAllRepositoriesStatesParams{
			IdFilter:    &uid,
			XApiVersion: v.conf.Veeam.XApiVersion,
		})
		if err != nil {
			return fmt.Errorf("could not get repositories states: %v", err)
		}

		var rep SingleRepository

		if err = json.NewDecoder(bytes.NewBuffer(rsr.Body)).Decode(&rep); err != nil {
			return fmt.Errorf("could not parse repositories states: %v", err)
		}

		v.AllRepositories = repos
		v.Repositories = append(v.Repositories, rep)
	}

	return nil
}

func (v *Veeam) GetProxies() error {
	v.log.Info("Collecting proxies information")
	resp, err := v.cl.GetAllProxiesWithResponse(v.ctx, &client.GetAllProxiesParams{
		XApiVersion: v.conf.Veeam.XApiVersion,
	})
	if err != nil {
		return fmt.Errorf("could not get proxies: %v", err)
	}

	var pr Proxies
	if err = json.NewDecoder(bytes.NewBuffer(resp.Body)).Decode(&pr); err != nil {
		return fmt.Errorf("could not parse proxies: %v", err)
	}

	v.Proxies = pr

	return nil
}

func (v *Veeam) GetBackupObjects() error {
	v.log.Info("Collecting backup objects information")

	resp, err := v.cl.GetAllBackupObjectsWithResponse(v.ctx, &client.GetAllBackupObjectsParams{
		XApiVersion: v.conf.Veeam.XApiVersion,
	})
	if err != nil {
		return fmt.Errorf("could not get backup objects: %v", err)
	}

	var bo BackupObjects
	if err = json.NewDecoder(bytes.NewBuffer(resp.Body)).Decode(&bo); err != nil {
		return fmt.Errorf("could not parse backup objects: %v", err)
	}

	v.BackupObjects = bo

	return nil
}
