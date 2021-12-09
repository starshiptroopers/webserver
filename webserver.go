package webserver

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var robotsUserAgent = []string{"facebook", "WhatsApp", "Viber", "TelegramBot", "Twitter", "Instagram", "Wget"}

type WebServerConfig struct {
	Logger     *zerolog.Logger
	LoggerHttp *zerolog.Logger
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
			//			c.Next()
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
		c.Set("robot", false)
		for _, rgxp := range regexps {
			if rgxp.MatchString(c.Request.UserAgent()) {
				c.Set("robot", true)
			}
		}
		c.Next()
	}
}

func (w WebServer) Run() {
	log := *(w.config.Logger)
	log.Info().Int("Port", w.config.Port).Msg("Starting listener")

	err := w.gin.Run(":" + strconv.Itoa(w.config.Port))

	if err != nil {
		log.Error().Msgf("webserver startup error: %v", err)
	}
}
