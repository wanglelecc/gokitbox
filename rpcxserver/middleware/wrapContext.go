package middleware

import "context"

type Point func(*WrapContext) error

type WrapContext struct {
	Methods   []Point
	Fn        Point
	index     int
	MethodTag string
	ctx       context.Context
}

func (w *WrapContext) Next() error {
	if w.index < len(w.Methods) {
		fn := w.Methods[w.index]
		w.index++
		return fn(w)
	}
	// 防止 Fn 被重复调用
	if w.index == len(w.Methods) {
		w.index++
		return w.Fn(w)
	}
	// 超出范围，说明中间件链已执行完毕
	return nil
}

func (w *WrapContext) Reset() {
	w.index = 0
	w.Methods = nil
	w.Fn = nil
	w.MethodTag = ""
	w.ctx = nil
}

func (w *WrapContext) SetCtx(c context.Context) {
	w.ctx = c
}

func (w *WrapContext) GetCtx() context.Context {
	return w.ctx
}
