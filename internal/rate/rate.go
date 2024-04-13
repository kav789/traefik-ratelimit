package rate

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Limiter struct {
	limit *int32
	win   []*int32
	close *int32
}

const WINLEN = 5

func NewLimiter(lim int) *Limiter {
	l := Limiter{
		limit: new(int32),
		close: new(int32),
		win:   make([]*int32, WINLEN),
	}
	for i := 0; i < WINLEN; i++ {
		l.win[i] = new(int32)
		*l.win[i] = int32(lim)
	}
	*l.limit = int32(lim)
	go func() {
		for atomic.LoadInt32(l.close) == 0 {
			i := time.Now().Second() % WINLEN
			atomic.StoreInt32(l.win[(i+(WINLEN/2)+WINLEN)%WINLEN], atomic.LoadInt32(l.limit))
			time.Sleep(time.Second)
		}
	}()
	return &l
}

func (l *Limiter) SetLimit(lim int) {
	atomic.StoreInt32(l.limit, int32(lim))
}

func (l *Limiter) Close() {
	atomic.StoreInt32(l.close, 1)
}

func (l *Limiter) Limit() int {
	return int(atomic.LoadInt32(l.limit))
}

func (l *Limiter) Allow() bool {
	//	t := atomic.LoadInt32(l.limit)
	i := time.Now().Second() % WINLEN
	//	atomic.StoreInt32(l.win[(i+(WINLEN/2)+WINLEN)%WINLEN], t)
	return atomic.AddInt32(l.win[i], -1) >= 0
}
