package grpcclient

import (
	"time"

	"google.golang.org/grpc"
)

// Parm 调用器配置项
type Parm struct {
	Name string

	PoolInitNum  int
	PoolCapacity int
	PoolIdle     time.Duration

	interceptors []grpc.UnaryClientInterceptor
}

// Option config wraps
type Option func(*Parm)

// WithPoolInitNum 连接池初始化数量
func WithPoolInitNum(num int) Option {
	return func(c *Parm) {
		c.PoolInitNum = num
	}
}

// WithPoolCapacity 连接池的容量大小
func WithPoolCapacity(num int) Option {
	return func(c *Parm) {
		c.PoolCapacity = num
	}
}

// WithPoolIdle 连接池的最大闲置时间
func WithPoolIdle(second int) Option {
	return func(c *Parm) {
		c.PoolIdle = time.Duration(second) * time.Second
	}
}

func AppendInterceptors(interceptor grpc.UnaryClientInterceptor) Option {
	return func(c *Parm) {
		c.interceptors = append(c.interceptors, interceptor)
	}
}
