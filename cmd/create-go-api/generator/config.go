package generator

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
	ProjectName string
	ModuleName  string
	Database    Database
	ORM         ORM
	Auth        AuthToken
	HasOAuth    bool
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
