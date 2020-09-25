package client

import (
	"context"
	"strings"

	"github.com/pojol/braid/module/linkcache"
)

// Builder 构建器接口
type Builder interface {
	// linker 是否引入链路缓存
	// bool 是否开启tracing
	Build(linker linkcache.ILinkCache, tracing bool) IClient
	Name() string
	SetCfg(cfg interface{}) error
}

// IClient rpc-client interface
type IClient interface {

	// ctx 链路的上下文，主要用于tracing
	// nodeName 逻辑节点名称, 用于查找目标节点地址
	// methon 方法名，用于定位到具体的rpc 执行函数
	// token 用户身份id
	// args request
	// reply result
	Invoke(ctx context.Context, nodeName, methon, token string, args, reply interface{})
}

var (
	m = make(map[string]Builder)
)

// Register regist rpc client
func Register(b Builder) {
	m[strings.ToLower(b.Name())] = b
}

// GetBuilder 获取构建器
func GetBuilder(name string) Builder {
	if b, ok := m[strings.ToLower(name)]; ok {
		return b
	}
	return nil
}
