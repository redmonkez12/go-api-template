package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFileName is the name of the JSON config file saved in generated projects.
const ConfigFileName = ".go-api-template.json"

// Database represents a supported database engine.
type Database string

const (
	DatabasePostgres Database = "postgres"
	DatabaseMySQL    Database = "mysql"
	DatabaseMongoDB  Database = "mongodb"
)

// ORM represents a supported ORM or database driver.
type ORM string

const (
	ORMBun    ORM = "bun"
	ORMGORM   ORM = "gorm"
	ORMPgx    ORM = "pgx"
	ORMSQLRaw ORM = "sqlraw"
	ORMMongo  ORM = "mongo"
)

// AuthToken represents a supported token strategy.
type AuthToken string

const (
	AuthPaseto AuthToken = "paseto"
	AuthJWT    AuthToken = "jwt"
)

// ProjectConfig holds all user selections for project generation.
type ProjectConfig struct {
	ProjectName string    `json:"project_name"`
	ModuleName  string    `json:"module_name"`
	Database    Database  `json:"database"`
	ORM         ORM       `json:"orm"`
	Auth        AuthToken `json:"auth"`
	HasOAuth    bool      `json:"has_oauth"`
}

// SaveToFile writes the config as JSON to ConfigFileName in the given directory.
func (c *ProjectConfig) SaveToFile(dir string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// LoadConfigFromFile reads ProjectConfig from ConfigFileName in the given directory.
func LoadConfigFromFile(dir string) (*ProjectConfig, error) {
	path := filepath.Join(dir, ConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	return &cfg, nil
}

// DatabaseLabel returns a human-readable label.
func (d Database) Label() string {
	switch d {
	case DatabasePostgres:
		return "PostgreSQL"
	case DatabaseMySQL:
		return "MySQL"
	case DatabaseMongoDB:
		return "MongoDB"
	default:
		return string(d)
	}
}

// ORMLabel returns a human-readable label.
func (o ORM) Label() string {
	switch o {
	case ORMBun:
		return "Bun"
	case ORMGORM:
		return "GORM"
	case ORMPgx:
		return "pgx (raw SQL)"
	case ORMSQLRaw:
		return "database/sql (raw)"
	case ORMMongo:
		return "mongo-go-driver"
	default:
		return string(o)
	}
}

// AuthLabel returns a human-readable label.
func (a AuthToken) Label() string {
	switch a {
	case AuthPaseto:
		return "PASETO v4"
	case AuthJWT:
		return "JWT (HS256)"
	default:
		return string(a)
	}
}
