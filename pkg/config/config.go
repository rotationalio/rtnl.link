package config

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rotationalio/go-ensign"
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
	Maintenance    bool                `default:"false" yaml:"maintenance"`
	Mode           string              `default:"release"`
	LogLevel       logger.LevelDecoder `split_words:"true" default:"info" yaml:"log_level"`
	ConsoleLog     bool                `split_words:"true" default:"false" yaml:"console_log"`
	BindAddr       string              `split_words:"true" default:":8765" yaml:"bind_addr"`
	AllowOrigins   []string            `split_words:"true" default:"http://localhost:8765"`
	Origin         string              `default:"https://rtnl.link"`
	AltOrigin      string              `split_words:"true" default:"https://r8l.co"`
	GoogleClientID string              `split_words:"true" required:"true"`
	AllowedDomain  string              `split_words:"true" default:"rotational.io"`
	Storage        StorageConfig
	Ensign         EnsignConfig
	processed      bool
	originURL      *url.URL
	altURL         *url.URL
}

type StorageConfig struct {
	ReadOnly bool   `split_words:"true" default:"false"`
	DataPath string `split_words:"true" required:"true"`
}

type EnsignConfig struct {
	Maintenance  bool   `env:"RTNL_MAINTENANCE"`
	Path         string `required:"false"`
	ClientID     string `split_words:"true"`
	ClientSecret string `split_words:"true"`
	Topic        string `default:"shortcrust-production"`
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

	if err = conf.Validate(); err != nil {
		return conf, err
	}

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

	if err = c.Ensign.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *Config) MakeOriginURLs(sid string) (link string, alt string) {
	if c.originURL == nil {
		c.originURL, _ = url.Parse(c.Origin)
	}

	if c.altURL == nil && c.AltOrigin != "" {
		c.altURL, _ = url.Parse(c.AltOrigin)
	}

	link = c.originURL.ResolveReference(&url.URL{Path: sid}).String()
	if c.altURL != nil {
		alt = c.altURL.ResolveReference(&url.URL{Path: sid}).String()
	}
	return link, alt
}

func (c EnsignConfig) Validate() error {
	if c.Maintenance {
		return nil
	}

	// Must have either the path specified or the client id and api key
	if c.Path == "" {
		if c.ClientID == "" || c.ClientSecret == "" {
			return ErrInvalidEnsignCredentials
		}
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		if c.Path == "" {
			return ErrInvalidEnsignCredentials
		}
	}

	return nil
}

func (c EnsignConfig) Options() ensign.Option {
	if c.Path != "" {
		return ensign.WithLoadCredentials(c.Path)
	}
	return ensign.WithCredentials(c.ClientID, c.ClientSecret)
}
