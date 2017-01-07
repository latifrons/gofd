package common

import (
	"errors"
	"fmt"
	"sync/atomic"

	log "github.com/cihub/seelog"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
)

// Service is a common service interface
type Service interface {
	Start() error
	Stop() bool
	OnStart(c *Config, e *echo.Echo) error
	OnStop(c *Config, e *echo.Echo)
	IsRunning() bool
}

// BaseService is the basic service struct with config and status
type BaseService struct {
	name    string
	running uint32 // atomic
	Cfg     *Config
	echo    *echo.Echo
	svc     Service
}

// NewBaseService return created a basic service instance
func NewBaseService(cfg *Config, name string, svc Service) *BaseService {
	return &BaseService{
		name:    name,
		running: 0,
		Cfg:     cfg,
		echo:    echo.New(),
		svc:     svc,
	}
}

// init log by config
func (s *BaseService) initlog() {
	if s.Cfg.Log != "" {
		if logger, err := log.LoggerFromConfigAsFile(s.Cfg.Log); err == nil {
			log.ReplaceLogger(logger)
		}
	}

	// init echo log
	s.echo.SetLogger(NewEchoLogger())
}

func (s *BaseService) runEcho() error {
	net := s.Cfg.Net
	var sr *standard.Server
	if net.TLS != nil {
		sr = standard.WithTLS(fmt.Sprintf("%s:%v", net.IP, net.MgntPort),
			net.TLS.Cert,
			net.TLS.Key,
		)
	} else {
		sr = standard.New(fmt.Sprintf("%s:%v", net.IP, net.MgntPort))
	}
	sr.SetHandler(s.echo)
	sr.SetLogger(s.echo.Logger())

	log.Infof("Starting http server %s:%v", net.IP, net.MgntPort)
	if err := sr.Start(); err != nil {
		log.Infof("Start http server %s:%v failed %v", net.IP, net.MgntPort, err)
		return err
	}
	return nil
}

// Start the service
func (s *BaseService) Start() error {
	if atomic.CompareAndSwapUint32(&s.running, 0, 1) {
		s.initlog()
		log.Infof("Starting %s", s.name)
		if err := s.svc.OnStart(s.Cfg, s.echo); err != nil {
			return err
		}
		go s.runEcho()
		return nil
	}
	return errors.New("Started aleadry.")
}

// OnStart implements Service
func (s *BaseService) OnStart(c *Config, e *echo.Echo) error { return nil }

// Stop the service
func (s *BaseService) Stop() bool {
	if atomic.CompareAndSwapUint32(&s.running, 1, 0) {
		log.Infof("Stopping %s", s.name)
		s.svc.OnStop(s.Cfg, s.echo)
		return true
	}
	return false
}

// OnStop implements Service
func (s *BaseService) OnStop(c *Config, e *echo.Echo) {}

// IsRunning implements Service
func (s *BaseService) IsRunning() bool {
	return atomic.LoadUint32(&s.running) == 1
}

// Auth using basic authorization
func (s *BaseService) Auth(u, p string) bool {
	if u == s.Cfg.Auth.Username && p == s.Cfg.Auth.Password {
		return true
	}
	return false
}
