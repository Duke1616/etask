package grpc

import (
	"fmt"
	"time"

	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"golang.org/x/net/context"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

const (
	initCapacityStr = "initCapacity"
	maxCapacityStr  = "maxCapacity"
	increaseStepStr = "increaseStep"
	growthRateStr   = "growthRate"
	nodeIDStr       = "nodeID"
)

type resolverBuilder struct {
	r       registry.Registry
	timeout time.Duration
}

func NewResolverBuilder(r registry.Registry, timeout time.Duration) resolver.Builder {
	return &resolverBuilder{
		r:       r,
		timeout: timeout,
	}
}

func (r *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	res := &executorResolver{
		target:       target,
		cc:           cc,
		registry:     r.r,
		close:        make(chan struct{}),
		updateNotify: make(chan struct{}, 1),
		timeout:      r.timeout,
	}
	go res.reconcileLoop()
	go res.watch()
	res.ResolveNow(resolver.ResolveNowOptions{})
	return res, nil
}

func (r *resolverBuilder) Scheme() string {
	return "executor"
}

type executorResolver struct {
	target        resolver.Target
	cc            resolver.ClientConn
	registry      registry.Registry
	close         chan struct{}
	updateNotify  chan struct{}
	timeout       time.Duration
	lastAddresses []resolver.Address
}

func (g *executorResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	select {
	case g.updateNotify <- struct{}{}:
	default:
	}
}

func (g *executorResolver) Close() {
	close(g.close)
}

func (g *executorResolver) watch() {
	events := g.registry.Subscribe(g.target.Endpoint())
	for {
		select {
		case <-events:
			g.ResolveNow(resolver.ResolveNowOptions{})
		case <-g.close:
			return
		}
	}
}

func (g *executorResolver) reconcileLoop() {
	for {
		select {
		case <-g.updateNotify:
			g.reconcile()
		case <-g.close:
			return
		}
	}
}

func (g *executorResolver) reconcile() {
	serviceName := g.target.Endpoint()
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	instances, err := g.registry.ListServices(ctx, serviceName)
	cancel()

	if err != nil {
		g.cc.ReportError(err)
		return
	}

	if len(instances) == 0 {
		g.cc.ReportError(fmt.Errorf("no endpoints found for service %s", serviceName))
		return
	}

	address := make([]resolver.Address, 0, len(instances))
	for _, ins := range instances {
		address = append(address, resolver.Address{
			Addr:       ins.Address,
			ServerName: ins.Name,
			Attributes: attributes.New(initCapacityStr, ins.InitCapacity).
				WithValue(maxCapacityStr, ins.MaxCapacity).
				WithValue(increaseStepStr, ins.IncreaseStep).
				WithValue(growthRateStr, ins.GrowthRate).
				WithValue(nodeIDStr, ins.ID),
		})
	}

	if isAddressesEqual(g.lastAddresses, address) {
		return
	}
	g.lastAddresses = address

	err = g.cc.UpdateState(resolver.State{
		Addresses: address,
	})
	if err != nil {
		g.cc.ReportError(err)
	}
}

func isAddressesEqual(a, b []resolver.Address) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]resolver.Address)
	for _, addr := range a {
		aMap[addr.Addr] = addr
	}
	for _, addr := range b {
		if _, ok := aMap[addr.Addr]; !ok {
			return false
		}
	}
	return true
}
