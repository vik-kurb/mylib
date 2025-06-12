package config

import (
	"time"
)

type BooksCacheConfig struct {
	Enable                  bool
	CleanupPeriod           time.Duration
	CleanupOldDataThreshold time.Duration
}
