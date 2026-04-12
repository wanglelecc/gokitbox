package middleware

import (
	"context"
	"sync"
)

type Middleware interface {
	WrapCall(ctx *WrapContext) error
}

type Options func() (Middleware, Point)

var (
	middlewares []Options
	mu          sync.RWMutex
)

// Use 注册全局中间件，线程安全
func Use(ops ...Options) {
	mu.Lock()
	defer mu.Unlock()
	middlewares = append(middlewares, ops...)
}

// getMiddlewares 获取当前所有中间件的副本，线程安全
func getMiddlewares() []Options {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Options, len(middlewares))
	copy(result, middlewares)
	return result
}

type RpcxService struct {
	Points      []Point
	Middlewares []Middleware
	ServiceName string
}

func (s *RpcxService) Init() {
	tempMiddleWares := make([]Middleware, 0)
	tempEndPoints := make([]Point, 0)

	// 使用线程安全的副本
	for _, op := range getMiddlewares() {
		middleware, endpoint := op()
		tempMiddleWares = append(tempMiddleWares, middleware)
		tempEndPoints = append(tempEndPoints, endpoint)
	}

	s.Middlewares = tempMiddleWares
	s.Points = tempEndPoints
}

func (s *RpcxService) WrapCall(ctx context.Context, tag string, end Point) (err error) {
	wrap := WrapContextPool.Get().(*WrapContext)
	wrap.Methods = s.Points
	wrap.Fn = end
	wrap.MethodTag = tag
	wrap.SetCtx(ctx)
	err = wrap.Next()
	wrap.Reset()
	WrapContextPool.Put(wrap)

	return
}

var WrapContextPool = sync.Pool{
	New: func() interface{} {
		return new(WrapContext)
	},
}
