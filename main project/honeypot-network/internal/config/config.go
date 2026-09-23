/*
©AngelaMos | 2026
config.go

Configuration loading and validation for the MirageNet deception network

Loads configuration from a YAML file with environment variable
overrides using the MIRAGE_ prefix. Each honeypot service has its
own sub-config controlling port, enabled state, and protocol-specific
options. Default() returns a fully populated config suitable for
local development.
*/

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sensor   SensorConfig   `yaml:"sensor"`
	SSH      SSHConfig      `yaml:"ssh"`
	HTTP     HTTPConfig     `yaml:"http"`
	FTP      FTPConfig      `yaml:"ftp"`
	SMB      SMBConfig      `yaml:"smb"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Redis    RedisConfig    `yaml:"redis"`
	Telnet   TelnetConfig   `yaml:"telnet"`
	Database DatabaseConfig `yaml:"database"`
	Stream   StreamConfig   `yaml:"stream"`
	API      APIConfig      `yaml:"api"`
	GeoIP    GeoIPConfig    `yaml:"geoip"`
	Log      LogConfig      `yaml:"log"`
}

type SensorConfig struct {
	ID       string `yaml:"id"`
	Hostname string `yaml:"hostname"`
	Region   string `yaml:"region"`
}

type SSHConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Port        int           `yaml:"port"`
	Banner      string        `yaml:"banner"`
	HostKeyPath string        `yaml:"host_key_path"`
	Hostname    string        `yaml:"hostname"`
	Timeout     time.Duration `yaml:"timeout"`
}

type HTTPConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Port         int    `yaml:"port"`
	ServerHeader string `yaml:"server_header"`
	TLSEnabled   bool   `yaml:"tls_enabled"`
	TLSPort      int    `yaml:"tls_port"`
}

type FTPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Banner  string `yaml:"banner"`
}

type SMBConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type MySQLConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Port          int    `yaml:"port"`
	ServerVersion string `yaml:"server_version"`
}

type RedisConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Port          int    `yaml:"port"`
	ServerVersion string `yaml:"server_version"`
}

type TelnetConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Port     int    `yaml:"port"`
	Banner   string `yaml:"banner"`
	Hostname string `yaml:"hostname"`
}

type DatabaseConfig struct {
	DSN            string        `yaml:"dsn"`
	PoolMin        int           `yaml:"pool_min"`
	PoolMax        int           `yaml:"pool_max"`
	ConnTimeout    time.Duration `yaml:"conn_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
	MigrationsPath string        `yaml:"migrations_path"`
}

type StreamConfig struct {
	URL       string `yaml:"url"`
	StreamKey string `yaml:"stream_key"`
	MaxLen    int64  `yaml:"max_len"`
	Password  string `yaml:"password"`
}

type APIConfig struct {
	Port         int           `yaml:"port"`
	CORSOrigins  []string      `yaml:"cors_origins"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type GeoIPConfig struct {
	DBPath string `yaml:"db_path"`
}

type LogConfig struct {
	Level      string `yaml:"level"`
	JSONFormat bool   `yaml:"json_format"`
	ReplayDir  string `yaml:"replay_dir"`
}

func Default() *Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = DefaultHostname
	}

	return &Config{
		Sensor: SensorConfig{
			ID:       "mirage-01",
			Hostname: hostname,
			Region:   "local",
		},
		SSH: SSHConfig{
			Enabled:     true,
			Port:        DefaultSSHPort,
			Banner:      SSHBanner,
			HostKeyPath: DefaultHostKeyPath,
			Hostname:    DefaultHostname,
			Timeout:     DefaultSessionTimeout,
		},
		HTTP: HTTPConfig{
			Enabled:      true,
			Port:         DefaultHTTPPort,
			ServerHeader: HTTPServer,
		},
		FTP: FTPConfig{
			Enabled: true,
			Port:    DefaultFTPPort,
			Banner:  FTPBanner,
		},
		SMB: SMBConfig{
			Enabled: true,
			Port:    DefaultSMBPort,
		},
		MySQL: MySQLConfig{
			Enabled:       true,
			Port:          DefaultMySQLPort,
			ServerVersion: MySQLBanner,
		},
		Redis: RedisConfig{
			Enabled:       true,
			Port:          DefaultRedisPort,
			ServerVersion: RedisBanner,
		},
		Telnet: TelnetConfig{
			Enabled:  true,
			Port:     DefaultTelnetPort,
			Banner:   TelnetBanner,
			Hostname: DefaultHostname,
		},
		Database: DatabaseConfig{
			DSN:            "postgres://miragenet:miragenet@localhost:5432/miragenet?sslmode=disable",
			PoolMin:        DefaultDBPoolMin,
			PoolMax:        DefaultDBPoolMax,
			ConnTimeout:    DefaultDBTimeout,
			IdleTimeout:    DefaultDBIdleTimeout,
			MigrationsPath: "migrations",
		},
		Stream: StreamConfig{
			URL:       "redis://localhost:16379",
			StreamKey: DefaultRedisStreamKey,
			MaxLen:    DefaultRedisMaxLen,
		},
		API: APIConfig{
			Port: DefaultAPIPort,
			CORSOrigins: []string{
				"http://localhost:5173",
				"http://localhost:3000",
			},
			ReadTimeout:  DefaultReadTimeout,
			WriteTimeout: DefaultWriteTimeout,
		},
		GeoIP: GeoIPConfig{
			DBPath: DefaultGeoIPPath,
		},
		Log: LogConfig{
			Level:      "info",
			JSONFormat: true,
			ReplayDir:  DefaultReplayDir,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err == nil {
		if parseErr := yaml.Unmarshal(data, cfg); parseErr != nil {
			return nil, fmt.Errorf(
				"parsing config %s: %w", path, parseErr,
			)
		}
	}

	applyEnvOverrides(cfg)

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("MIRAGE_SENSOR_ID"); v != "" {
		cfg.Sensor.ID = v
	}
	if v := os.Getenv("MIRAGE_SENSOR_REGION"); v != "" {
		cfg.Sensor.Region = v
	}
	if v := os.Getenv("MIRAGE_DATABASE_URL"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("MIRAGE_REDIS_URL"); v != "" {
		cfg.Stream.URL = v
	}
	if v := os.Getenv("MIRAGE_REDIS_PASSWORD"); v != "" {
		cfg.Stream.Password = v
	}
	if v := os.Getenv("MIRAGE_GEOIP_DB_PATH"); v != "" {
		cfg.GeoIP.DBPath = v
	}
	if v := os.Getenv("MIRAGE_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("MIRAGE_REPLAY_DIR"); v != "" {
		cfg.Log.ReplayDir = v
	}
	if v := os.Getenv("MIRAGE_CORS_ORIGINS"); v != "" {
		cfg.API.CORSOrigins = strings.Split(v, ",")
	}

	applyPortOverride("MIRAGE_SSH_PORT", &cfg.SSH.Port)
	applyPortOverride("MIRAGE_HTTP_PORT", &cfg.HTTP.Port)
	applyPortOverride("MIRAGE_FTP_PORT", &cfg.FTP.Port)
	applyPortOverride("MIRAGE_SMB_PORT", &cfg.SMB.Port)
	applyPortOverride("MIRAGE_MYSQL_PORT", &cfg.MySQL.Port)
	applyPortOverride("MIRAGE_REDIS_PORT", &cfg.Redis.Port)
	applyPortOverride("MIRAGE_TELNET_PORT", &cfg.Telnet.Port)
	applyPortOverride("MIRAGE_API_PORT", &cfg.API.Port)

	applyBoolOverride("MIRAGE_SSH_ENABLED", &cfg.SSH.Enabled)
	applyBoolOverride("MIRAGE_HTTP_ENABLED", &cfg.HTTP.Enabled)
	applyBoolOverride("MIRAGE_FTP_ENABLED", &cfg.FTP.Enabled)
	applyBoolOverride("MIRAGE_SMB_ENABLED", &cfg.SMB.Enabled)
	applyBoolOverride("MIRAGE_MYSQL_ENABLED", &cfg.MySQL.Enabled)
	applyBoolOverride("MIRAGE_REDIS_ENABLED", &cfg.Redis.Enabled)
	applyBoolOverride("MIRAGE_TELNET_ENABLED", &cfg.Telnet.Enabled)
}

func applyPortOverride(envKey string, target *int) {
	if v := os.Getenv(envKey); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			*target = port
		}
	}
}

func applyBoolOverride(envKey string, target *bool) {
	if v := os.Getenv(envKey); v != "" {
		*target = v == "true" || v == "1"
	}
}

func (c *Config) Addr(port int) string {
	return fmt.Sprintf(":%d", port)
}
