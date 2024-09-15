package mid

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/ardanlabs/service/business/web/v1/metrics"
	"github.com/ardanlabs/service/foundation/web"
)

// Panics recovers from panics and converts the panic to an error so it is
// reported in Metrics and handled in Errors.
func Panics() web.Middleware {
	m := func(handler web.Handler) web.Handler {
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {

			// Defer a function to recover from a panic and set the err return
			// variable after the fact.
			// 所以这里设置panic 为了 capture handler可能发生的panic ,具体的上下文我们可以在org roam 节点所以为了防止我们在处理http handler过程中发生panic,我们应该明确recover中查看
			defer func() {
				if rec := recover(); rec != nil {
					trace := debug.Stack()
					err = fmt.Errorf("PANIC [%v] TRACE[%s]", rec, string(trace))

					metrics.AddPanics(ctx)
				}
			}()

			return handler(ctx, w, r)
		}

		return h
	}

	return m
}
