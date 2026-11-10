// Package config loads and persists the application configuration.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// User is an SMB account served by the application's Samba instance.
type User struct {
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
	ReadOnly bool   `json:"readonly"`
	Enabled  bool   `json:"enabled"`
}

// Remote is an external share (usually a Proxmox host) that is mounted and
// re-exported inside the aggregated tree.
type Remote struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Share    string `json:"share"`
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	SubPath  string `json:"subpath,omitempty"`
	Enabled  bool   `json:"enabled"`
	ReadOnly bool   `json:"readonly"`
	Options  string `json:"options,omitempty"`
}

// SMB holds settings of the managed Samba instance.
type SMB struct {
	Enabled       bool     `json:"enabled"`
	ListenIP      string   `json:"listen_ip"`
	BindDevice    string   `json:"bind_device"`
	Workgroup     string   `json:"workgroup"`
	ServerString  string   `json:"server_string"`
	ShareName     string   `json:"share_name"`
	SharePath     string   `json:"share_path"`
	ShareReadOnly bool     `json:"share_readonly"`
	ForceUser     bool     `json:"force_user"`
	Ports         string   `json:"ports"`
	MinProtocol   string   `json:"min_protocol"`
	MaxProtocol   string   `json:"max_protocol"`
	Users         []User   `json:"users"`
	GuestOK       bool     `json:"guest_ok"`
	ExtraOptions  []string `json:"extra_options,omitempty"`
}

// Web holds HTTP UI settings.
type Web struct {
	Listen   string `json:"listen"`
	User     string `json:"user"`
	Password string `json:"password"`
	ReadOnly bool   `json:"readonly"`
}

// Stats holds sampling preferences.
type Stats struct {
	IntervalSec int `json:"interval_sec"`
	HistoryLen  int `json:"history_len"`
}

// Paths describes runtime locations (not user editable via UI).
type Paths struct {
	ConfigFile string `json:"-"`
	DataDir    string `json:"-"`
	SambaConf  string `json:"samba_conf"`
	SambaPidDir string `json:"samba_pid_dir"`
}

// Config is the root document persisted to disk.
type Config struct {
	Web     Web     `json:"web"`
	SMB     SMB     `json:"smb"`
	Remotes []Remote `json:"remotes"`
	Stats   Stats   `json:"stats"`
	Paths   Paths   `json:"paths"`
}

// mu serialises Save/Update across goroutines without living inside Config
// (so Config stays copyable and JSON-encodable without vet warnings).
var mu sync.Mutex

// Default returns configuration suitable for a fresh install.
func Default() *Config {
	return &Config{
		Web: Web{
			Listen:   "0.0.0.0:8099",
			User:     "admin",
			Password: "lip368lip",
		},
		SMB: SMB{
			Enabled:      true,
			ListenIP:     "192.168.160.2",
			BindDevice:   "br0",
			Workgroup:    "WORKGROUP",
			ServerString: "keenetic-smb-ui",
			ShareName:    "all",
			SharePath:    "/opt/var/keenetic-smb-ui/share",
			Ports:        "445",
			ForceUser:    true,
			MinProtocol:  "SMB2",
			MaxProtocol:  "SMB3_11",
			Users: []User{
				{Name: "admin", Password: "lip368lip", Enabled: true},
			},
		},
		Stats: Stats{IntervalSec: 5, HistoryLen: 120},
		Paths: Paths{
			SambaConf:   "/opt/etc/keenetic-smb-ui/smb.conf",
			SambaPidDir: "/opt/var/keenetic-smb-ui/run",
		},
	}
}

// Load reads the configuration file; a missing file yields defaults.
func Load(path string) (*Config, error) {
	cfg := Default()
	cfg.Paths.ConfigFile = path
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Configs written before force_user existed default to enabling it: without
	// it ordinary Samba users cannot write into the root-owned aggregated tree.
	if !strings.Contains(string(data), `"force_user"`) {
		cfg.SMB.ForceUser = true
	}
	cfg.normalize()
	return cfg, nil
}

func (c *Config) normalize() {
	def := Default()
	if c.Web.Listen == "" {
		c.Web.Listen = def.Web.Listen
	}
	if c.Web.User == "" {
		c.Web.User = def.Web.User
	}
	if c.SMB.ListenIP == "" {
		c.SMB.ListenIP = def.SMB.ListenIP
	}
	if c.SMB.ShareName == "" {
		c.SMB.ShareName = def.SMB.ShareName
	}
	if c.SMB.SharePath == "" {
		c.SMB.SharePath = def.SMB.SharePath
	}
	if c.SMB.Workgroup == "" {
		c.SMB.Workgroup = def.SMB.Workgroup
	}
	if c.SMB.MinProtocol == "" {
		c.SMB.MinProtocol = def.SMB.MinProtocol
	}
	if c.SMB.MaxProtocol == "" {
		c.SMB.MaxProtocol = def.SMB.MaxProtocol
	}
	if c.SMB.Ports == "" {
		c.SMB.Ports = def.SMB.Ports
	}
	if c.Paths.SambaConf == "" {
		c.Paths.SambaConf = def.Paths.SambaConf
	}
	if c.Paths.SambaPidDir == "" {
		c.Paths.SambaPidDir = def.Paths.SambaPidDir
	}
	if c.Stats.IntervalSec <= 0 {
		c.Stats.IntervalSec = def.Stats.IntervalSec
	}
	if c.Stats.HistoryLen <= 0 {
		c.Stats.HistoryLen = def.Stats.HistoryLen
	}
	if c.Paths.ConfigFile == "" {
		c.Paths.ConfigFile = def.Paths.ConfigFile
	}
}

// Save writes the configuration atomically.
func (c *Config) Save() error {
	mu.Lock()
	defer mu.Unlock()
	return c.saveLocked()
}

func (c *Config) saveLocked() error {
	path := c.Paths.ConfigFile
	if path == "" {
		path = "/opt/etc/keenetic-smb-ui/config.json"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Update applies fn to a copy of the config and persists it on success.
func (c *Config) Update(fn func(*Config) error) error {
	mu.Lock()
	defer mu.Unlock()
	snapshot := *c
	if err := fn(&snapshot); err != nil {
		return err
	}
	snapshot.normalize()
	if err := snapshot.saveLocked(); err != nil {
		return err
	}
	*c = snapshot
	return nil
}
