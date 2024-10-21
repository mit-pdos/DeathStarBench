package shardclnt

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
)

func init() {}

const (
	SHARD_REGISTER_PORT = ":9998"
)

type ShardClnt[T any] struct {
	mu       sync.Mutex
	clnts    []*T
	newShard func(string) *T
	idx      int
}

func NewShardClnt[T any](newShard func(string) *T) *ShardClnt[T] {
	c := &ShardClnt[T]{
		clnts:    []*T{},
		newShard: newShard,
	}

	c.startRPCServer()

	return c
}

func (sc *ShardClnt[T]) AddClnt(clnt *T) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.addClntL(clnt)
}

func (sc *ShardClnt[T]) addClntL(clnt *T) {
	sc.clnts = append(sc.clnts, clnt)
}

func (sc *ShardClnt[T]) GetRR() (*T, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if len(sc.clnts) == 0 {
		return nil, fmt.Errorf("No shards registered")
	}

	sc.idx++
	idx := sc.idx % len(sc.clnts)

	return sc.clnts[idx], nil
}

type RegisterShardRequest struct {
	Addr string
}

type RegisterShardResponse struct {
	OK bool
}

func (sc *ShardClnt[T]) RegisterShard(req *RegisterShardRequest) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	log.Printf("Registering new shard %v", req.Addr)
	sc.addClntL(sc.newShard(req.Addr))
	log.Printf("Done registering new shard %v", req.Addr)
	return nil
}

type ShardClntWrapper struct {
	addClntFn func(req *RegisterShardRequest)
}

func (scw *ShardClntWrapper) RegisterShard(req *RegisterShardRequest, rep *RegisterShardResponse) error {
	scw.addClntFn(req)
	rep.OK = true
	return nil
}

func (c *ShardClnt[T]) startRPCServer() {
	scw := &ShardClntWrapper{
		func(req *RegisterShardRequest) {
			c.RegisterShard(req)
		},
	}
	rpc.Register(scw)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", SHARD_REGISTER_PORT)
	if err != nil {
		log.Fatalf("Error Listen in Coordinator.registerServer: %v", err)
	}
	go http.Serve(l, nil)
}
