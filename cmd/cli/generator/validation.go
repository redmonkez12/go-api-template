package generator

import "fmt"

// validCombinations defines which DB+ORM pairings are supported.
var validCombinations = map[Database][]ORM{
	DatabasePostgres: {ORMBun, ORMGORM, ORMPgx},
	DatabaseMySQL:    {ORMGORM, ORMBun, ORMSQLRaw},
	DatabaseMongoDB:  {ORMMongo},
}

// ValidateConfig checks that the ProjectConfig has a valid DB+ORM combination
// and all required fields are set.
func ValidateConfig(cfg *ProjectConfig) error {
	if cfg.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}
	if cfg.ModuleName == "" {
		return fmt.Errorf("module name is required")
	}

	allowed, ok := validCombinations[cfg.Database]
	if !ok {
		return fmt.Errorf("unsupported database: %s", cfg.Database)
	}

	for _, orm := range allowed {
		if orm == cfg.ORM {
			return nil
		}
	}

	return fmt.Errorf("invalid combination: %s + %s", cfg.Database.Label(), cfg.ORM.Label())
}

// ORMsForDatabase returns the valid ORM choices for a given database.
func ORMsForDatabase(db Database) []ORM {
	return validCombinations[db]
}
