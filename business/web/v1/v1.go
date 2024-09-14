// Package v1 manages the different versions of the API.
package v1

import (
	"net/http"
	"os"

	"github.com/ardanlabs/service/business/web/v1/auth"
	"github.com/ardanlabs/service/business/web/v1/mid"
	"github.com/ardanlabs/service/foundation/logger"
	"github.com/ardanlabs/service/foundation/web"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
)

// Options represent optional parameters.
type Options struct {
	corsOrigin string
}

// WithCORS provides configuration options for CORS.
func WithCORS(origin string) func(opts *Options) {
	return func(opts *Options) {
		opts.corsOrigin = origin
	}
}

// APIMuxConfig contains all the mandatory systems required by handlers.
type APIMuxConfig struct {
	UsingWeaver bool
	Build       string
	Shutdown    chan os.Signal
	Log         *logger.Logger
	Auth        *auth.Auth
	DB          *sqlx.DB
	Tracer      trace.Tracer
}

// RouteAdder defines behavior that sets the routes to bind for an instance
// of the service.
// RouteAdder 定义了为服务实例设置绑定路由的行为。
// 服务的实例绑定路由的行为。
type RouteAdder interface {
	Add(app *web.App, cfg APIMuxConfig)
}

// APIMux constructs a http.Handler with all application routes defined.
func APIMux(cfg APIMuxConfig, routeAdder RouteAdder, options ...func(opts *Options)) http.Handler {
	var opts Options
	for _, option := range options {
		option(&opts)
	}
	// 这里有个细节,我们仅仅是注册Middleware,并没有执行wrap,这里每个实例返回的是Middleware
	app := web.NewApp(
		cfg.Shutdown,
		cfg.Tracer,
		mid.Logger(cfg.Log),
		mid.Errors(cfg.Log),
		mid.Metrics(),
		mid.Panics(),
	)

	if opts.corsOrigin != "" {
		app.EnableCORS(mid.Cors(opts.corsOrigin))
	}

	routeAdder.Add(app, cfg)

	return app
}
