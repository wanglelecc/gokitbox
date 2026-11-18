package consumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/wanglelecc/gokitbox/taskqueue"
	"sync/atomic"
	"time"
)

func (r *consumerRuntime) consumeReliable(ctx context.Context, quit <-chan struct{}, client *redis.Client, source, kind, failed string, retries int, tpl string, callback ConsumeCallback, successes, failures *int64) (clean bool) {
	defer func() {
		if p := recover(); p != nil {
			r.recordError(fmt.Errorf("reliable consumer panic (%T)", p))
			clean = false
		}
	}()
	visibility := r.opts().VisibilityTimeout
	q, err := taskqueue.New(client, source, kind, visibility)
	if err != nil {
		r.recordError(err)
		return false
	}
	for !stopRequested(ctx, quit) {
		operation, cancel := boundedContext(ctx, r.opts().OperationTimeout)
		m, err := q.Claim(operation)
		cancel()
		if errors.Is(err, redis.Nil) {
			if !waitRetry(ctx, quit, 100*time.Millisecond) {
				return true
			}
			continue
		}
		if err != nil {
			r.recordError(err)
			return stopRequested(ctx, quit)
		}
		callCtx, stop := context.WithCancel(taskqueue.WithTaskID(ctx, m.ID))
		renewDone := make(chan struct{})
		renewErr := make(chan error, 1)
		go func() {
			defer close(renewDone)
			ticker := time.NewTicker(visibility / 3)
			defer ticker.Stop()
			for {
				select {
				case <-callCtx.Done():
					return
				case <-ticker.C:
					operation, cancel := boundedContext(callCtx, r.opts().OperationTimeout)
					err := q.Renew(operation, m)
					cancel()
					if err != nil {
						renewErr <- err
						stop()
						return
					}
				}
			}
		}()
		ok := false
		for attempt := 0; attempt <= retries && !stopRequested(callCtx, quit); attempt++ {
			if r.call(callCtx, callback, tpl, source, m.Payload) {
				ok = true
				break
			}
			if attempt < retries && !waitRetry(callCtx, quit, r.opts().RetryDelay) {
				break
			}
		}
		stop()
		<-renewDone
		select {
		case err := <-renewErr:
			r.recordError(err)
			return stopRequested(ctx, quit)
		default:
		}
		operation, cancel = boundedContext(context.WithoutCancel(ctx), r.opts().OperationTimeout)
		if ok {
			err = q.Ack(operation, m)
			if err == nil {
				atomic.AddInt64(successes, 1)
			}
		} else if stopRequested(ctx, quit) {
			err = q.Retry(operation, m)
		} else {
			err = q.Fail(operation, m, failed)
			if err == nil {
				atomic.AddInt64(failures, 1)
			}
		}
		cancel()
		if err != nil {
			r.recordError(err)
			return stopRequested(ctx, quit)
		}
	}
	return true
}
