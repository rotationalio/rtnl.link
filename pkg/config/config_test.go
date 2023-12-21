package config_test

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var testEnv = map[string]string{
	"RTNL_MAINTENANCE":          "true",
	"RTNL_MODE":                 "test",
	"RTNL_LOG_LEVEL":            "debug",
	"RTNL_CONSOLE_LOG":          "true",
	"RTNL_BIND_ADDR":            ":8888",
	"RTNL_ALLOW_ORIGINS":        "http://localhost:8888",
	"RTNL_ORIGIN":               "http://localhost:8888",
	"RTNL_ALT_ORIGIN":           "http://127.0.0.1:8888",
	"RTNL_STORAGE_READ_ONLY":    "true",
	"RTNL_STORAGE_DATA_PATH":    "/data/db",
	"RTNL_ENSIGN_PATH":          "/credentials/ensign.json",
	"RTNL_ENSIGN_CLIENT_ID":     "ensignclientid",
	"RTNL_ENSIGN_CLIENT_SECRET": "ensignclientsecret",
	"RTNL_ENSIGN_TOPIC":         "shortcrust-testing",
}

func TestConfig(t *testing.T) {
	// Set required environment variables and cleanup after the test is complete.
	t.Cleanup(cleanupEnv())
	setEnv()

	conf, err := config.New()
	require.NoError(t, err, "could not process configuration from the environment")
	require.False(t, conf.IsZero(), "processed config should not be zero valued")

	// Ensure configuration is correctly set from the environment
	require.True(t, conf.Maintenance)
	require.Equal(t, gin.TestMode, conf.Mode)
	require.Equal(t, zerolog.DebugLevel, conf.GetLogLevel())
	require.True(t, conf.ConsoleLog)
	require.Equal(t, testEnv["RTNL_BIND_ADDR"], conf.BindAddr)
	require.Equal(t, []string{testEnv["RTNL_ALLOW_ORIGINS"]}, conf.AllowOrigins)
	require.Equal(t, testEnv["RTNL_ORIGIN"], conf.Origin)
	require.Equal(t, testEnv["RTNL_ALT_ORIGIN"], conf.AltOrigin)
	require.True(t, conf.Storage.ReadOnly)
	require.Equal(t, testEnv["RTNL_STORAGE_DATA_PATH"], conf.Storage.DataPath)
	require.Equal(t, testEnv["RTNL_ENSIGN_PATH"], conf.Ensign.Path)
	require.Equal(t, testEnv["RTNL_ENSIGN_CLIENT_ID"], conf.Ensign.ClientID)
	require.Equal(t, testEnv["RTNL_ENSIGN_CLIENT_SECRET"], conf.Ensign.ClientSecret)
	require.Equal(t, testEnv["RTNL_ENSIGN_TOPIC"], conf.Ensign.Topic)

	// Ensure the sentry release is correctly set
	// require.True(t, strings.HasPrefix(conf.Sentry.GetRelease(), "rtnl@"))
}

func TestEnsignValidation(t *testing.T) {
	testCases := []struct {
		conf config.EnsignConfig
		err  error
	}{
		{
			config.EnsignConfig{},
			config.ErrInvalidEnsignCredentials,
		},
		{
			config.EnsignConfig{ClientID: "foo"},
			config.ErrInvalidEnsignCredentials,
		},
		{
			config.EnsignConfig{ClientSecret: "foo"},
			config.ErrInvalidEnsignCredentials,
		},
		{
			config.EnsignConfig{Path: "credentials.json"},
			nil,
		},
		{
			config.EnsignConfig{ClientID: "foo", ClientSecret: "bar"},
			nil,
		},
		{
			config.EnsignConfig{Path: "zap", ClientID: "foo", ClientSecret: "bar"},
			nil,
		},
	}

	for i, tc := range testCases {
		err := tc.conf.Validate()
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err, "test case %d failed", i)
		} else {
			require.NoError(t, err, "test case %d failed", i)
		}
	}
}

// Returns the current environment for the specified keys, or if no keys are specified
// then it returns the current environment for all keys in the testEnv variable.
func curEnv(keys ...string) map[string]string {
	env := make(map[string]string)
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := os.LookupEnv(key); ok {
				env[key] = val
			}
		}
	} else {
		for key := range testEnv {
			env[key] = os.Getenv(key)
		}
	}

	return env
}

// Sets the environment variables from the testEnv variable. If no keys are specified,
// then this function sets all environment variables from the testEnv.
func setEnv(keys ...string) {
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := testEnv[key]; ok {
				os.Setenv(key, val)
			}
		}
	} else {
		for key, val := range testEnv {
			os.Setenv(key, val)
		}
	}
}

// Cleanup helper function that can be run when the tests are complete to reset the
// environment back to its previous state before the test was run.
func cleanupEnv(keys ...string) func() {
	prevEnv := curEnv(keys...)
	return func() {
		for key, val := range prevEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
