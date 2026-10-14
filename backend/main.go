package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"automarket/audit"
	"automarket/auth"
	"automarket/models"
	"automarket/service"
	"automarket/storage"
)

const (
	listenAddr        = "localhost:8080"
	dataDir           = "data"
	adminName         = "Administrador"
	sessionTTL        = time.Hour
	minAuditKeyLength = 32
)

// config reúne los secretos, que se reciben por variables de entorno para
// no dejarlos escritos en el código.
type config struct {
	auditKey      string
	adminEmail    string
	adminPassword string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	users, err := storage.OpenTable[models.User](filepath.Join(dataDir, "users.json"))
	if err != nil {
		return fmt.Errorf("abrir usuarios: %w", err)
	}
	vehicles, err := storage.OpenTable[models.Vehicle](filepath.Join(dataDir, "vehicles.json"))
	if err != nil {
		return fmt.Errorf("abrir vehículos: %w", err)
	}
	logger, err := audit.Open(filepath.Join(dataDir, "audit.log"), []byte(cfg.auditKey))
	if err != nil {
		return fmt.Errorf("abrir bitácora: %w", err)
	}
	defer logger.Close()

	vehicleService := service.NewVehicles(vehicles, users)
	accountService := service.NewAccounts(users, vehicleService, auth.NewSessions(sessionTTL))
	if err := accountService.EnsureAdmin(service.RegisterInput{
		Name:     adminName,
		Email:    cfg.adminEmail,
		Password: cfg.adminPassword,
	}); err != nil {
		return fmt.Errorf("crear administrador: %w", err)
	}

	router, err := newRouter(logger, accountService, vehicleService)
	if err != nil {
		return err
	}
	log.Printf("AutoMarket escuchando en http://%s", listenAddr)
	return router.Run(listenAddr)
}

func loadConfig() (config, error) {
	cfg := config{
		auditKey:      os.Getenv("AUTOMARKET_AUDIT_KEY"),
		adminEmail:    os.Getenv("AUTOMARKET_ADMIN_EMAIL"),
		adminPassword: os.Getenv("AUTOMARKET_ADMIN_PASSWORD"),
	}
	if len(cfg.auditKey) < minAuditKeyLength {
		return cfg, fmt.Errorf("AUTOMARKET_AUDIT_KEY debe tener al menos %d caracteres", minAuditKeyLength)
	}
	if cfg.adminEmail == "" || cfg.adminPassword == "" {
		return cfg, errors.New("AUTOMARKET_ADMIN_EMAIL y AUTOMARKET_ADMIN_PASSWORD son obligatorias")
	}
	return cfg, nil
}
