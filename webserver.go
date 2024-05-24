package webserver

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var robotsUserAgent = []string{"facebook", "WhatsApp", "Viber", "TelegramBot", "Twitter", "Instagram", "Wget"}

// InitTimeout defines a webserver initialization timeout,
// a maximum duration the RunBg method is blocked at
var InitTimeout = time.Millisecond * 100

type WebServerConfig struct {
	Logger     *zerolog.Logger
	LoggerHttp *zerolog.Logger
	Addr       string
	Port       int
}

type globalState struct {
	sync.Mutex
	requestCounter uint64
}

type WebServer struct {
	config    WebServerConfig
	gin       *gin.Engine
	altRoutes []iRoute
	state     globalState
	srv       *http.Server // is only used in gorouting startup mode
}

type iRoute struct {
	Path    *regexp.Regexp
	Method  string
	Handler func(ctx *gin.Context)
}

func NewWebServer(config WebServerConfig) (*WebServer, error) {

	gin.SetMode(gin.ReleaseMode)
	webServer := &WebServer{
		config: config,
		gin:    gin.New(),
		state: globalState{
			requestCounter: 0,
		},
	}

	webServer.gin.Use(
		func(c *gin.Context) {
			webServer.state.Lock()
			webServer.state.requestCounter++
			//set requestID
			c.Set("requestID", webServer.state.requestCounter)
			webServer.state.Unlock()
			c.Next()
		},
	)

	webServer.gin.Use(webServer.httpLogger(config.LoggerHttp))
	webServer.gin.Use(webServer.robotsDetect(robotsUserAgent))
	webServer.gin.Use(gin.Recovery())

	webServer.gin.NoRoute(webServer.AltRouter)
	return webServer, nil
}

func (w *WebServer) ServiceRegister(group string, services ...WebService) {
	var router *gin.RouterGroup
	//create group if defined
	if group != "" {
		router = w.gin.Group(group)
	} else {
		router = w.gin.Group("/")
	}

	for _, s := range services {
		//some service related initalization
		if err := s.Init(w.gin); err != nil {
			w.config.Logger.Error().Err(err).Msg("Can't initialize web service")
		}
		//register service middlewares
		for _, h := range s.Middlewares() {
			router.Use(h)
		}
		//register service's handlers
		for _, route := range s.GinRoutes() {
			router.Handle(route.Method, route.Path, route.Handler)
		}

		//register service's alternative routes described with regexp (regexp isn't supported by gin)
		for _, route := range s.AltRoutes() {
			w.altRoutes = append(
				w.altRoutes,
				iRoute{
					regexp.MustCompile(route.Path),
					route.Method,
					func(c *gin.Context) {
						for _, h := range s.Middlewares() {
							h(c)
						}
						route.Handler(c)
					},
				})
		}
	}
}

func (w *WebServer) AltRouter(c *gin.Context) {
	for _, route := range w.altRoutes {
		if route.Path.MatchString(c.Request.RequestURI) {
			route.Handler(c)
			return
		}
	}
}

func (w *WebServer) httpLogger(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestID uint64

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		if v, ok := c.Get("requestID"); ok {
			if requestID, ok = v.(uint64); !ok {
				requestID = 0
			}
		}

		c.Get("requestID")
		// Process request
		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		if _, exists := c.Get("httpNoLogging"); exists {
			return
		}

		logger.Info().
			Int64("latency", time.Now().Sub(start).Milliseconds()).
			Str("clientIp", c.ClientIP()).
			Str("path", path).
			Str("method", c.Request.Method).
			Int("statusCode", c.Writer.Status()).
			Int("bodySize", c.Writer.Size()).
			Uint64("requestID", requestID).
			Msg("http request")

	}
}

func (w *WebServer) robotsDetect(names []string) gin.HandlerFunc {
	var regexps []*regexp.Regexp

	for _, name := range names {
		regexps = append(regexps, regexp.MustCompile("(?i)"+name))
	}
	return func(c *gin.Context) {
		if c.GetHeader("X-Robot") != "" {
			c.Set("robot", true)
		} else {
			c.Set("robot", false)
			for _, rgxp := range regexps {
				if rgxp.MatchString(c.Request.UserAgent()) {
					c.Set("robot", true)
				}
			}
		}
		c.Next()
	}
}

// Run runs a gin server,
// this method will block the calling goroutine indefinitely unless an error happens.
func (w WebServer) Run() {
	log := *(w.config.Logger)
	log.Info().Str("Addr", w.config.Addr).Int("Port", w.config.Port).Msg("Starting listener")

	err := w.gin.Run(w.bindTo(w.config.Addr, w.config.Port))

	if err != nil {
		log.Error().Msgf("webserver startup error: %v", err)
	}
}

// RunBg runs a gin server in goroutine and exits immediately
// on server success init or InitTimeout happened,
func (w *WebServer) RunBg() (err error) {
	log := *(w.config.Logger)
	log.Info().Str("Addr", w.config.Addr).Int("Port", w.config.Port).Msg("Starting listener")

	w.srv = &http.Server{
		Addr:    w.bindTo(w.config.Addr, w.config.Port),
		Handler: w.gin.Handler(),
	}

	startupError := make(chan error)
	go func() {
		e := w.srv.ListenAndServe()
		if e != http.ErrServerClosed {
			startupError <- e
		}
	}()

	select {
	case <-time.After(InitTimeout):
	case err = <-startupError:
	}

	if err != nil {
		log.Error().Msgf("webserver startup error: %v", err)
		err = fmt.Errorf("can't start web server: %w", err)
	} else {
		log.Info().Msgf("webserver was started and listen on %v", w.srv.Addr)
	}
	return
}

// Shutdown performs gracefully shutdown of a server started with RunBg
func (w *WebServer) Shutdown(ctx context.Context) (err error) {
	if w.srv != nil {
		err = w.srv.Shutdown(ctx)
		w.config.Logger.Info().Msg("webserver shutdown")
	}
	return nil
}

func (w WebServer) bindTo(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}
