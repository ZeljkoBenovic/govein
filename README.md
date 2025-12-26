# GOVEIN (GO-VEeam-INfluxdb)
A Veeam Backup & Replication metrics exporter.     
An evolution of Jorge's great work https://github.com/jorgedlcruz/veeam-backup-and-replication-grafana.    
Re-written in GO for better adoption and ease of use, mainly in containerized environments.


## Requirements
* InfluxDB 2.0+
* Veeam B&R 12+

### InfluxDB create table
InfluxDB version must be greater than 2.0.     
Bucket must be created before running the exporter.
```bash
influx bucket create --name <bucket_name> --org <organization_name_or_id> --retention <duration> --token <your_token>
```

## Config file
YAML config file is used to configure the exporter. A config file starter can be quickly created with `govein -export`, 
then customize it to fit your needs.    
Config file example
```yaml
# veeam server config
veeam:
  # veeam api
  host: https://veeam.server:9419
  # veeam api version
  x_api_version: 1.2-rev0
  # if veeam has self-signed cert set to true
  trust_self_signed_cert: false
  # veeam admin username - can be set using env var
  username: <veeam-admin or VEEAM_ADMIN_USERNAME env var>
  # veeam admin password - can be set using env var
  password: <veeam-admin-password or VEEAM_ADMIN_PASSWORD env var>
  # excluded job types
  excluded_job_types:
    MalwareDetection: {}
    SecurityComplianceAnalyzer: {}
# influxdb config
influx:
  # influxdb api
  host: http://influxdb:8086
  # influxdb token - can be set using env var
  token: <influxdb-token or INFLUXDB_TOKEN env var>
  # influxdb organisation - can be set using env var
  org: <influxdb-org-name or INFLUXDB_ORG_NAME env var>
  # influxdb bucket - must be prepared in advance on influxdb server
  bucket: veeam(must be created)
# log level (INFO, DEBUG, ERROR)
log_level: INFO
# scrape interval
interval_seconds: 1800
```

Once config file is set, start the exporter with `govein -config ./config.yaml`. 
Scraping process will repeat on a specified time interval, one hour by default.

## Secrets management
In containerized environments secrets are usually injected via environment variables, which `govein` supports.   
* Use `VEEAM_ADMIN_USERNAME` instead of `veeam.username` in the config file 
* Use `VEEAM_ADMIN_PASSWORD` instead of `veeam.password` in the config file 
* Use `INFLUXDB_TOKEN` instead of `influx.token` in the config file 
* Use `INFLUXDB_ORG` instead of `influx.org` in the config file

## Grafana
A sample dashboard can be found in the `examples` folder, which is entirely a work of Jorge.    
He also has a great writeup on how to connect this together 
https://jorgedelacruz.uk/2023/05/31/looking-for-the-perfect-dashboard-influxdb-telegraf-and-grafana-part-xliv-monitoring-veeam-backup-replication-api/, 
but instead of `veeam_backup_and_replication.sh` use this exporter. 

## Helm Chart
* Add repo `helm repo add govein https://zeljkobenovic.github.io/govein`
* Install using `helm install my-govein govein/govein`
* Uninstall with `helm uninstall my-govein`

## License
MIT