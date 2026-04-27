package service

import (
	providerpkg "dappco.re/go/api/pkg/provider"
	"github.com/gin-gonic/gin"
)

type stubDaemonProvider struct {
	name     string
	basePath string
	channels []string
}

func (p stubDaemonProvider) Name() string { return p.name }

func (p stubDaemonProvider) BasePath() string { return p.basePath }

func (p stubDaemonProvider) RegisterRoutes(_ *gin.RouterGroup) {}

func (p stubDaemonProvider) Channels() []string {
	return append([]string(nil), p.channels...)
}

var _ providerpkg.Streamable = stubDaemonProvider{}

type stubRouteGroup struct {
	name     string
	basePath string
}

func (g stubRouteGroup) Name() string { return g.name }

func (g stubRouteGroup) BasePath() string { return g.basePath }

func (g stubRouteGroup) RegisterRoutes(_ *gin.RouterGroup) {}
