package webserver

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

type PublicWebService struct {
	log    *zerolog.Logger
	router *gin.Engine
}

func (s *PublicWebService) Init(router *gin.Engine) error {
	s.router = router
	return nil
}

func (s PublicWebService) GinRoutes() []WebRoute {
	return []WebRoute{
		{Path: "/",
			Method:  "GET",
			Handler: s.indexHandler},
	}
}

func (s PublicWebService) AltRoutes() []WebRoute {
	return []WebRoute{}
}

func (s PublicWebService) Middlewares() []func(ctx *gin.Context) {
	return []func(ctx *gin.Context){}
}

func (s PublicWebService) indexHandler(ctx *gin.Context) {
	ctx.String(200, "HELLO")
}

func TestWebServer_Run(t *testing.T) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.StampMicro}).With().Timestamp().Logger()
	service := &PublicWebService{
		&logger,
		nil,
	}

	webServerConfig := WebServerConfig{
		Logger:     &logger,
		LoggerHttp: &logger,
		Port:       9091,
	}

	webServer, err := NewWebServer(webServerConfig)
	if err != nil {
		t.Fatal(err)
	}

	webServer.ServiceRegister("", service)

	go webServer.Run()

	time.Sleep(time.Second)

	client := &http.Client{}

	resp, err := client.Get("http://localhost:9091")
	if err != nil {
		log.Fatalf("Failed get: %s", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if string(body) != "HELLO" {
		t.Fatalf("Wrong answer: %v", string(body))
	}
	fmt.Println()
}
