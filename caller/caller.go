package caller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/pojol/braid/link"

	"github.com/pojol/braid/consul"
	"github.com/pojol/braid/log"

	"github.com/opentracing/opentracing-go"
	"github.com/pojol/braid/utility"

	"github.com/pojol/braid/cache/pool"
	"github.com/pojol/braid/caller/brpc"
	"github.com/pojol/braid/tracer"
	"google.golang.org/grpc"
)

type (
	// Caller 调用器
	Caller struct {
		coordinateAddress string

		cfg Config

		poolMgr sync.Map
		sync.Mutex
	}

	// Config 调用器配置项
	Config struct {
		ConsulAddress string

		PoolInitNum  int
		PoolCapacity int
		PoolIdle     time.Duration

		Tracing bool
	}
)

var (
	defaultConfig = Config{
		ConsulAddress: "http://127.0.0.1:8500",
		PoolInitNum:   8,
		PoolCapacity:  32,
		PoolIdle:      time.Second * 120,
		Tracing:       false,
	}
	c *Caller

	// ErrBoxUnavailiable 没有可用的box
	ErrBoxUnavailiable = errors.New("caller box unavailable")
	// ErrConfigConvert 配置转换失败
	ErrConfigConvert = errors.New("Convert linker config")
)

// New 构建指针
func New() *Caller {
	c = &Caller{}
	return c
}

// Init 通过配置构建调用器
func (c *Caller) Init(cfg interface{}) error {
	callerCfg, ok := cfg.(Config)
	if !ok {
		return ErrConfigConvert
	}

	proxy := ""
	services, err := consul.GetCatalogServices(callerCfg.ConsulAddress, "coordinate")
	if err != nil {
		return err
	}

	if len(services) == 0 {
		log.Fatalf(ErrBoxUnavailiable.Error())
	} else {
		proxys := []string{}
		for k := range services {
			proxys = append(proxys, k)
		}
		idx := utility.RandSpace(0, int64(len(proxys)-1))
		proxy = proxys[idx]
	}

	address := services[proxy].ServiceAddress + ":" + strconv.Itoa(services[proxy].ServicePort)
	fmt.Println("discover coordinate", address)

	c.coordinateAddress = address
	c.cfg = callerCfg

	return nil
}

// Get get caller global pointer
func Get() *Caller {
	return c
}

// Run run
func (c *Caller) Run() {

}

// Close 释放调用器
func (c *Caller) Close() {

}

// Call 执行一次rpc调用
func (c *Caller) Call(parentCtx context.Context, boxName string, serviceName string, token string, body []byte) (res *brpc.RouteRes, err error) {

	var address string
	var caPool *pool.GRPCPool
	var caConn *pool.ClientConn
	var caCtx context.Context
	var caCancel context.CancelFunc
	var connCtx context.Context
	var connCancel context.CancelFunc
	var method string
	res = new(brpc.RouteRes)

	c.Lock()
	defer c.Unlock()

	address, err = c.findBox(parentCtx, boxName, serviceName, token)
	if err != nil {
		goto EXT
	}

	caPool, err = c.pool(address)
	if err != nil {
		goto EXT
	}

	connCtx, connCancel = context.WithTimeout(parentCtx, time.Second)
	defer connCancel()
	caConn, err = caPool.Get(connCtx)
	if err != nil {
		goto EXT
	}
	defer caConn.Put()

	caCtx, caCancel = context.WithTimeout(parentCtx, time.Second)
	defer caCancel()

	method = "/brpc.gateway/routing"
	err = caConn.Invoke(caCtx, method, &brpc.RouteReq{
		Box:     boxName,
		Service: serviceName,
		ReqBody: body,
	}, res)
	if err != nil {
		caConn.Unhealthy()
		goto EXT
	}

EXT:
	if err != nil {
		log.SysError("caller", "do", err.Error())
	}

	return res, err
}

// FindBox 通过查找器获取目标box
func (c *Caller) findBox(parentCtx context.Context, boxName string, serviceName string, key string) (string, error) {
	var address string
	var err error

	if key != "" {

		address, err = link.Get().Target(key)
		if err != nil {
			goto EXT
		}

		if address != "" {
			goto EXT
		}

		address, err = c.getBoxWithCoordinate(parentCtx, boxName, serviceName)
		if err != nil {
			goto EXT
		}

		link.Get().Link(key, address)
	} else {
		address, err = c.getBoxWithCoordinate(parentCtx, boxName, serviceName)
		if err != nil {
			goto EXT
		}
	}

EXT:
	if err != nil {
		// log
		log.SysError("caller", "findBox", err.Error())
	}

	return address, err
}

// Pool 获取grpc连接池
func (c *Caller) pool(address string) (p *pool.GRPCPool, err error) {
	factory := func() (*grpc.ClientConn, error) {
		var conn *grpc.ClientConn
		var err error

		if c.cfg.Tracing {
			interceptor := tracer.ClientInterceptor(opentracing.GlobalTracer())
			conn, err = grpc.Dial(address, grpc.WithInsecure(), grpc.WithUnaryInterceptor(interceptor))
		} else {
			conn, err = grpc.Dial(address)
		}

		if err != nil {
			return nil, err
		}

		return conn, nil
	}

	pi, ok := c.poolMgr.Load(address)
	if !ok {
		pool, err := pool.NewGRPCPool(factory, c.cfg.PoolInitNum, c.cfg.PoolCapacity, c.cfg.PoolIdle)
		if err != nil {
			goto EXT
		}
		c.poolMgr.Store(address, pool)
		pi = pool
	}

	p = pi.(*pool.GRPCPool)

EXT:
	if err != nil {
		log.SysError("caller", "pool", err.Error())
	}
	return p, err
}

func (c *Caller) getBoxWithCoordinate(parentCtx context.Context, boxName string, serviceName string) (string, error) {
	rres := new(brpc.RouteRes)
	var fres struct {
		Address string
	}
	var address string
	var caCtx context.Context
	var caCancel context.CancelFunc
	var dat []byte
	var conn *pool.ClientConn
	method := "/brpc.gateway/routing"

	p, err := c.pool(c.coordinateAddress)
	if err != nil {
		goto EXT
	}
	conn, err = p.Get(context.Background())
	if err != nil {
		goto EXT
	}
	defer conn.Put()

	caCtx, caCancel = context.WithTimeout(parentCtx, time.Second)
	defer caCancel()

	dat, _ = json.Marshal(struct {
		Box     string
		Service string
	}{boxName, serviceName})

	if conn.Invoke(caCtx, method, &brpc.RouteReq{
		Box:     "coordinate",
		Service: "find",
		ReqBody: dat,
	}, rres) != nil {
		goto EXT
	}

	json.Unmarshal(rres.ResBody, &fres)
	address = fres.Address

EXT:
	if err != nil {
		log.SysError("caller", "getBoxWithCoordinate", err.Error())
	}

	return address, err
}