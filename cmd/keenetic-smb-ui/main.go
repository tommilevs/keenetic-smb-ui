// Command keenetic-smb-ui is a self-contained SMB server manager + web UI for
// Keenetic routers running Entware. It supervises Samba (opkg samba4-server),
// binds it to a dedicated IP, aggregates Proxmox/SMB shares into one tree and
// exposes configuration, statistics and logs through an embedded web UI.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tommilevs/keenetic-smb-ui/internal/config"
	"github.com/tommilevs/keenetic-smb-ui/internal/web"
)

var version = "dev"

func main() {
	var (
		cfgPath  = flag.String("config", "/opt/etc/keenetic-smb-ui/config.json", "path to config file")
		listen   = flag.String("listen", "", "override web listen address, e.g. 0.0.0.0:8099")
		dataDir  = flag.String("data", "/opt/var/keenetic-smb-ui", "state directory")
		showVer  = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("keenetic-smb-ui %s\n", version)
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *listen != "" {
		cfg.Web.Listen = *listen
	}
	cfg.Paths.ConfigFile = *cfgPath
	cfg.Paths.DataDir = *dataDir

	srv, err := web.New(cfg)
	if err != nil {
		log.Fatalf("web: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run() }()
	log.Printf("keenetic-smb-ui %s listening on http://%s", version, cfg.Web.Listen)
	srv.Bootstrap()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("server: %v", err)
		}
	case s := <-sig:
		log.Printf("signal %s, shutting down", s)
		shutdownErr := srv.Shutdown(5 * time.Second)
		if shutdownErr != nil {
			log.Printf("shutdown: %v", shutdownErr)
		}
	}
}
