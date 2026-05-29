package config

import "github.com/kelseyhightower/envconfig"

type Settings struct {
	Port             int    `envconfig:"SCOREOPS_PORT" default:"8090"`
	DatabaseURL      string `envconfig:"SCOREOPS_DATABASE_URL" required:"true"`
	DBPoolMax        int32  `envconfig:"SCOREOPS_DB_POOL_MAX" default:"4"`
	InternalSecret   string `envconfig:"SCOREOPS_INTERNAL_SECRET" required:"true"`
	PillarMinWeight  int    `envconfig:"SCOREOPS_PILLAR_MIN_WEIGHT" default:"5"`
	PillarMaxWeight  int    `envconfig:"SCOREOPS_PILLAR_MAX_WEIGHT" default:"60"`
	LogLevel         string `envconfig:"SCOREOPS_LOG_LEVEL" default:"info"`
	TitlisAPIURL     string `envconfig:"SCOREOPS_TITLISAPI_URL"`
	SkipMigrations   bool   `envconfig:"SCOREOPS_SKIP_MIGRATIONS" default:"false"`
}

func Load() (*Settings, error) {
	var s Settings
	if err := envconfig.Process("", &s); err != nil {
		return nil, err
	}
	return &s, nil
}
