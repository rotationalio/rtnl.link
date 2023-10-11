package config

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rotationalio/rtnl.link/pkg/logger"
	"github.com/rs/zerolog"
)

// All environment variables will have this prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be RTNL_LOG_LEVEL
// because of this prefix and the split_words struct tag in the conf below.
const prefix = "rtnl"

// Config contains all of the configuration parameters for an rtnl server and is
// loaded from the environment or a configuration file with reasonable defaults for
// values that are omitted. The Config should be validated in preparation for running
// the server to ensure that all server operations work as expected.
type Config struct {
	Maintenance  bool                `default:"false" yaml:"maintenance"`
	Mode         string              `default:"release"`
	LogLevel     logger.LevelDecoder `split_words:"true" default:"info" yaml:"log_level"`
	ConsoleLog   bool                `split_words:"true" default:"false" yaml:"console_log"`
	BindAddr     string              `split_words:"true" default:":8765" yaml:"bind_addr"`
	AllowOrigins []string            `split_words:"true" default:"http://localhost:8765"`
	Storage      StorageConfig
	processed    bool
}

type StorageConfig struct {
	ReadOnly bool   `split_words:"true" default:"false"`
	DataPath string `split_words:"true" required:"true"`
}

// New creates and processes a Config from the environment ready for use. If the
// configuration is invalid or it cannot be processed an error is returned.
func New() (conf Config, err error) {
	if err = confire.Process(prefix, &conf); err != nil {
		return conf, err
	}

	// Ensure the Sentry release is set to rtnl.
	// if conf.Sentry.Release == "" {
	// 	conf.Sentry.Release = fmt.Sprintf("rtnl@%s", pkg.Version())
	// }

	conf.processed = true
	return conf, nil
}

// Parse and return the zerolog log level for configuring global logging.
func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}

// A Config is zero-valued if it hasn't been processed by a file or the environment.
func (c Config) IsZero() bool {
	return !c.processed
}

// Mark a manually constructed config as processed as long as its valid.
func (c Config) Mark() (Config, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	c.processed = true
	return c, nil
}

// Validates the config is ready for use in the application and that configuration
// semantics such as requiring multiple required configuration parameters are enforced.
func (c Config) Validate() (err error) {
	if c.Mode != gin.ReleaseMode && c.Mode != gin.DebugMode && c.Mode != gin.TestMode {
		return fmt.Errorf("invalid configuration: %q is not a valid gin mode", c.Mode)
	}
	return nil
}
