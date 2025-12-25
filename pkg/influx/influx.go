package influx

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ZeljkoBenovic/govein/pkg/config"
	"github.com/ZeljkoBenovic/govein/pkg/veeam"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Influx struct {
	ctx  context.Context
	log  *slog.Logger
	cl   influxdb2.Client
	conf config.Config
	wb   api.WriteAPIBlocking
}

func NewInflux(ctx context.Context, conf config.Config, log *slog.Logger) (*Influx, error) {
	cl := influxdb2.NewClient(conf.Influx.Host, conf.Influx.Token)

	resp, err := cl.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("influx db server not healthy: %v", err)
	}

	log.Info("InfluxDB server info",
		"name", resp.Name,
		"status", resp.Status,
		"msg", *resp.Message,
		"version", *resp.Version,
	)

	return &Influx{
		ctx:  ctx,
		log:  log,
		cl:   cl,
		conf: conf,
		wb:   cl.WriteAPIBlocking(conf.Influx.Org, conf.Influx.Bucket),
	}, nil
}

func (i *Influx) SetVeeamServerInfo(info veeam.ServerInfo) error {
	i.log.Info("Storing veeam server info into database")

	p := influxdb2.NewPointWithMeasurement("veeam_vbr_info").
		AddTag("veeamVBRId", info.VbrId).
		AddTag("veeamVBRName", info.Name).
		AddTag("veeamVBRVersion", info.BuildVersion).
		AddTag("veeamVBR", i.conf.Veeam.Host).
		AddTag("veeamDatabaseVendor", info.DatabaseVendor).
		AddField("vbr", 1)

	return i.wb.WritePoint(i.ctx, p)
}

func (i *Influx) SetVeeamSessions(sess veeam.Sessions) error {
	i.log.Info("Storing sessions into database")

	result := map[string]int{
		"Success": 1,
		"Warning": 2,
		"Failed":  3,
	}

	for _, s := range sess.Data {
		if s.Result.Result == "None" {
			i.log.Debug("Skipping session with no data", "session_name", s.Name)
			continue
		}

		if _, ok := i.conf.Veeam.ExcludedJobTypes[s.SessionType]; ok {
			i.log.Debug("Skipping session with excluded job type", "session_name", s.Name, "session_type", s.SessionType)
			continue
		}

		p := influxdb2.NewPointWithMeasurement("veeam_vbr_sessions").
			AddTag("veeamVBR", i.conf.Veeam.Host).
			AddTag("veeamVBRSessionJobName", s.Name).
			AddTag("veeamVBRSessiontype", s.SessionType).
			AddTag("veeamVBRSessionsJobState", s.State).
			AddTag("veeamVBRSessionsJobResultMessage", s.Result.Message).
			AddField("veeamVBRSessionsJobResult", result[s.Result.Result]).
			AddField("veeamBackupSessionsTimeDuration", s.EndTime.Sub(s.CreationTime).Seconds()).
			SetTime(s.EndTime)

		if err := i.wb.WritePoint(i.ctx, p); err != nil {
			return fmt.Errorf("could not write veeam session: %v", err)
		}
	}

	return nil
}

func (i *Influx) SetManagedServers(srv veeam.ManagedSevers) error {
	i.log.Info("Storing managed servers into database")

	for ind, s := range srv.Data {
		p := influxdb2.NewPointWithMeasurement("veeam_vbr_managedservers").
			AddTag("veeamVBR", i.conf.Veeam.Host).
			AddTag("veeamVBRMSName", s.Name).
			AddTag("veeamVBRMStype", s.Type).
			AddTag("veeamVBRMSDescription", s.Description).
			AddField("veeamVBRMSInternalID", ind)

		if err := i.wb.WritePoint(i.ctx, p); err != nil {
			return fmt.Errorf("could not write veeam managed servers: %v", err)
		}
	}
	return nil
}

func (i *Influx) SetRepositories(v veeam.Veeam) error {
	i.log.Info("Storing repositories into database")
	boolToString := map[bool]string{
		true:  "true",
		false: "false",
	}

	for _, r := range v.Repositories {
		for _, rd := range r.Data {
			for _, a := range v.AllRepositories.Data {
				if rd.ID == a.ID {
					p := influxdb2.NewPointWithMeasurement("veeam_vbr_repositories").
						AddTag("veeamVBR", i.conf.Veeam.Host).
						AddTag("veeamVBRRepoName", rd.Name).
						AddTag("veeamVBRRepoType", rd.Type).
						AddTag("veeamVBRMSDescription", rd.Description)

					switch rd.Type {
					case "WinLocal", "Nfs", "Smb":
						p.AddTag("veeamVBRRepopath", strings.TrimRight(rd.Path, "\\"))
						p.AddTag("veeamVBRRepoPerVM", boolToString[a.Repository.AdvancedSettings.PerVMBackup])
						p.AddField("veeamVBRRepoMaxtasks", a.Repository.MaxTaskCount)
						p.AddField("veeamVBRRepoCapacity", rd.CapacityGB*1024*1024*1024)
						p.AddField("veeamVBRRepoFree", rd.FreeGB*1024*1024*1024)
						p.AddField("veeamVBRRepoUsed", rd.UsedSpaceGB*1024*1024*1024)
						// TODO: support more types
					default:
						i.log.Error("Unknown repository type", "type", rd.Type)
					}

					if err := i.wb.WritePoint(i.ctx, p); err != nil {
						return fmt.Errorf("could not write veeam repositories: %v", err)
					}

					if err := i.wb.Flush(i.ctx); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (i *Influx) SetProxies(pr veeam.Proxies) error {
	i.log.Info("Storing proxies into database")

	for _, p := range pr.Data {
		data := influxdb2.NewPointWithMeasurement("veeam_vbr_proxies").
			AddTag("veeamVBR", i.conf.Veeam.Host).
			AddTag("veeamVBRProxyName", p.Name).
			AddTag("veeamVBRProxyType", p.Type).
			AddTag("veeamVBRProxyDescription", p.Description).
			AddTag("veeamVBRProxyMode", p.Server.TransportMode).
			AddField("veeamVBRProxyTask", p.Server.MaxTaskCount)

		if err := i.wb.WritePoint(i.ctx, data); err != nil {
			return fmt.Errorf("could not write veeam proxies: %v", err)
		}
	}

	return nil
}

func (i *Influx) SetBackupObjects(bo veeam.BackupObjects) error {
	i.log.Info("Storing backup objects into database")

	for _, b := range bo.Data {
		p := influxdb2.NewPointWithMeasurement("veeam_vbr_backupobjects").
			AddTag("veeamVBR", i.conf.Veeam.Host).
			AddTag("veeamVBRBobjectName", b.Name).
			AddTag("veeamVBRBobjecttype", string(b.Type)).
			AddTag("veeamVBRBobjectPlatform", string(b.PlatformName)).
			AddTag("veeamVBRBobjectviType", string(b.ViType)).
			AddTag("veeamVBRBobjectObjectId", b.ObjectID).
			AddTag("veeamVBRBobjectPath", b.Path).
			AddField("restorePointsCount", b.RestorePointsCount)

		if err := i.wb.WritePoint(i.ctx, p); err != nil {
			return fmt.Errorf("could not write veeam backup objects: %v", err)
		}
	}

	return nil
}

func (i *Influx) FlushAndClose() error {
	if err := i.wb.Flush(i.ctx); err != nil {
		return err
	}
	i.cl.Close()
	return nil
}
