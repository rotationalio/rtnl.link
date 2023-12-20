package rtnl

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rotationalio/rtnl.link/pkg/logger"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = logger.GCPFieldKeyTime
	zerolog.MessageFieldName = logger.GCPFieldKeyMsg
	zerolog.DurationFieldInteger = false
	zerolog.DurationFieldUnit = time.Millisecond

	// Add the severity hook for GCP logging
	var gcpHook logger.SeverityHook
	log.Logger = zerolog.New(os.Stdout).Hook(gcpHook).With().Timestamp().Logger()
}

// Implements the link shortening service and API.
type Server struct {
	sync.RWMutex
	conf    config.Config   // Primary source of truth for server configuration
	srv     *http.Server    // The HTTP server configuration for handling requests
	router  *gin.Engine     // The gin router for mapping endpoints to handlers
	db      storage.Storage // Database storage for URLs and API keys
	healthy bool            // Indicates that the service is online and healthy
	ready   bool            // Indicates that the service is ready to accept requests
	started time.Time       // The timestamp that the server was started (for uptime)
	url     *url.URL        // The endpoint that the server is hosted on
	echan   chan error      // Sending errors down this channel stops the server (is fatal)
}

func New(conf config.Config) (s *Server, err error) {
	// Load the default configuration from the environment if the config is empty.
	if conf.IsZero() {
		if conf, err = config.New(); err != nil {
			return nil, err
		}
	}

	// Setup our logging config first thing
	zerolog.SetGlobalLevel(conf.GetLogLevel())
	if conf.ConsoleLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Create and configure the gin router
	gin.SetMode(conf.Mode)
	router := gin.New()
	router.RedirectTrailingSlash = true
	router.RedirectFixedPath = false
	router.HandleMethodNotAllowed = true
	router.ForwardedByClientIP = true
	router.UseRawPath = false
	router.UnescapePathValues = true

	// Create the http server
	srv := &http.Server{
		Addr:              conf.BindAddr,
		Handler:           router,
		ErrorLog:          nil,
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	s = &Server{
		conf:   conf,
		srv:    srv,
		router: router,
		echan:  make(chan error, 1),
	}

	return s, nil
}

func (s *Server) Serve() (err error) {
	// Setup database connection
	if !s.conf.Maintenance {
		if s.db, err = storage.Open(s.conf.Storage); err != nil {
			return err
		}
	}

	// Setup routes and middleware
	if err = s.Routes(s.router); err != nil {
		return err
	}

	// Catch OS signals for graceful shutdowns
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.echan <- s.Shutdown(ctx)
	}()

	// Create a socket to listen on and infer the final URL.
	// NOTE: if the bindaddr is 127.0.0.1:0 for testing, a random port will be assigned,
	// manually creating the listener will allow us to determine which port.
	// When we start listening all incoming requests will be buffered until the server
	// actually starts up in its own go routine below.
	var sock net.Listener
	if sock, err = net.Listen("tcp", s.srv.Addr); err != nil {
		return fmt.Errorf("could not listen on bind addr %s: %s", s.srv.Addr, err)
	}

	// Set the URL from the listener and indcate the server has started
	// NOTE: no locking is necessary since no requests can come in
	s.setURL(sock.Addr())
	s.started = time.Now()
	s.healthy = true

	// Listen for HTTP requests and handle them
	go func() {
		if err := s.srv.Serve(sock); !errors.Is(err, http.ErrServerClosed) {
			s.echan <- err
		}
		s.echan <- nil
	}()

	s.SetReady(true)
	log.Info().Str("listen", s.URL()).Str("version", pkg.Version()).Msg("rtnl server started")
	return <-s.echan
}

func (s *Server) Shutdown(ctx context.Context) (err error) {
	// Set ready to false to prevent additional requests
	s.SetReady(false)
	defer s.SetHealthy(false)

	s.srv.SetKeepAlivesEnabled(false)
	if serr := s.srv.Shutdown(ctx); serr != nil {
		err = errors.Join(err, serr)
	}

	if serr := s.db.Close(); serr != nil {
		err = errors.Join(err, serr)
	}

	return err
}

func (s *Server) Routes(router *gin.Engine) (err error) {
	// Kubernetes probe endpoints -- add routes before middleware to ensure that these
	// endpoints are not logged nor subject to other handling.
	router.GET("/healthz", s.Healthz)
	router.GET("/livez", s.Healthz)
	router.GET("/readyz", s.Readyz)

	// Setup HTML template renderer
	includes := []string{"templates/layout/*.html", "templates/partials/*.html"}
	if s.router.HTMLRender, err = NewRender(content, "templates/*.html", includes...); err != nil {
		return err
	}

	// Setup CORS configuration
	corsConf := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-CSRF-TOKEN", "sentry-trace", "baggage"},
		AllowCredentials: true,
		AllowOrigins:     s.conf.AllowOrigins,
		MaxAge:           12 * time.Hour,
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger("rtnl"),

		// Panic recovery middleware
		gin.Recovery(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(corsConf),

		// Maintenance mode handling - should not require authentication
		s.Available(),
	}

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			router.Use(middleware)
		}
	}

	// Setup Static Content Server
	var static fs.FS
	if static, err = fs.Sub(content, "static"); err != nil {
		return err
	}
	router.StaticFS("/static", http.FS(static))

	// Add the v1 API routes
	v1 := router.Group("/v1")
	{
		// Heartbeat route (no authentication required)
		v1.GET("/status", s.Status)
		v1.POST("/shorten", s.Authenticate, s.ShortenURL)
	}

	// Permenant Routes
	router.GET("/:id", s.Redirect)
	router.DELETE("/:id", s.Authenticate, s.DeleteShortURL)
	router.GET("/:id/info", s.Authenticate, s.ShortURLInfo)

	// Web Routes
	// TODO: add authentication
	router.GET("/", s.Index)

	// NotFound and NotAllowed routes
	router.NoRoute(s.NotFound)
	router.NoMethod(s.NotAllowed)
	return nil
}

// Set the URL from the TCPAddr when the server is started.
func (s *Server) setURL(addr net.Addr) {
	s.url = &url.URL{Scheme: "http", Host: addr.String()}
	if s.srv.TLSConfig != nil {
		s.url.Scheme = "https"
	}

	if tcp, ok := addr.(*net.TCPAddr); ok && tcp.IP.IsUnspecified() {
		s.url.Host = fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}
}

// URL returns the URL of the server determined by the socket addr.
func (s *Server) URL() string {
	s.RLock()
	defer s.RUnlock()
	return s.url.String()
}

// Compute how long the server has been running for status calls.
func (s *Server) Uptime() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return time.Since(s.started)
}
