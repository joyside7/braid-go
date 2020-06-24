package k8selector

import (
	"context"
	"errors"
	"time"

	"github.com/pojol/braid/3rd/log"
	"github.com/pojol/braid/module/election"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	// ElectionName 基于kubernetes接口实现的选举器
	ElectionName = "K8sElector"
)

var (
	// ErrConfigConvert 配置转换失败
	ErrConfigConvert = errors.New("convert config error")
)

type k8sElectorBuilder struct{}

func newK8sElector() election.Builder {
	return &k8sElectorBuilder{}
}

func onStartedLeading(ctx context.Context) {

}

func onStoppedLeading() {

}

func getConfig(cfg string) (*rest.Config, error) {
	if cfg == "" {
		return rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", cfg)
}

func newClientset(filename string) (*kubernetes.Clientset, error) {
	config, err := getConfig(filename)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (*k8sElectorBuilder) Name() string {
	return ElectionName
}

func (*k8sElectorBuilder) Build(cfg interface{}) (election.IElection, error) {
	kecfg, ok := cfg.(Cfg)
	if !ok {
		return nil, ErrConfigConvert
	}

	clientset, err := newClientset(kecfg.KubeCfg)
	if err != nil {
		return nil, err
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "braid-lock",
			Namespace: kecfg.Namespace,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: kecfg.NodID,
		},
	}

	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   30 * time.Second,  // 租约时间
		RenewDeadline:   10 * time.Second,  // 更新租约时间
		RetryPeriod:     kecfg.RetryPeriod, // 非master节点的重试时间
		Callbacks: leaderelection.LeaderCallbacks{
			OnNewLeader: func(identity string) {
				if identity == kecfg.NodID {
					log.SysElection(identity)
				}
			},
		},
	})
	if err != nil {
		return nil, err
	}

	el := &k8sElector{
		cfg:     kecfg,
		elector: elector,
	}

	return el, nil
}

// Cfg k8s elector config
type Cfg struct {
	KubeCfg     string
	NodID       string
	Namespace   string
	RetryPeriod time.Duration
}

type k8sElector struct {
	cfg     Cfg
	lock    *resourcelock.LeaseLock
	elector *leaderelection.LeaderElector
}

func (e *k8sElector) IsMaster() bool {
	return e.elector.IsLeader()
}

func (e *k8sElector) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.elector.Run(ctx)
}

func (e *k8sElector) Close() {

}

func init() {
	election.Register(newK8sElector())
}
