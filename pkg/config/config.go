package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rotationalio/rtnl.link/pkg/logger"
	"github.com/rs/zerolog"
)

// All environment variables will have this Prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be RTNL_LOG_LEVEL
// because of this Prefix and the split_words struct tag in the conf below.
const Prefix = "rtnl"

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
	Origin       string              `default:"https://rtnl.link"`
	AltOrigin    string              `split_words:"true" default:"https://r8l.co"`
	Storage      StorageConfig
	Auth         AuthConfig
	processed    bool
	originURL    *url.URL
	altURL       *url.URL
}

type StorageConfig struct {
	ReadOnly bool   `split_words:"true" default:"false"`
	DataPath string `split_words:"true" required:"true"`
}

type AuthConfig struct {
	GoogleClientID  string            `split_words:"true" required:"true" desc:"the Google oauth claims client id and audience"`
	HDClaim         string            `split_words:"true" default:"rotational.io" desc:"the email domain to allow to authenticate"`
	CookieDomain    string            `split_words:"true" default:"rtnl.link" desc:"the domain to assign cookies to"`
	Keys            map[string]string `required:"false" desc:"rsa keys for signing access tokens (generated if omitted)"`
	Audience        string            `default:"https://rtnl.link" desc:"audience to add to rtnl jwt claims"`
	Issuer          string            `default:"https://rtnl.link" desc:"issuer to add to rtnl jwt claims"`
	AccessDuration  time.Duration     `split_words:"true" default:"1h" desc:"amount of time access tokens are valid"`
	RefreshDuration time.Duration     `split_words:"true" default:"2h" desc:"amount of time refresh tokens are valid"`
	RefreshOverlap  time.Duration     `split_words:"true" default:"-15m" desc:"validity period of refresh token while access token is"`
}

// New creates and processes a Config from the environment ready for use. If the
// configuration is invalid or it cannot be processed an error is returned.
func New() (conf Config, err error) {
	if err = confire.Process(Prefix, &conf); err != nil {
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
