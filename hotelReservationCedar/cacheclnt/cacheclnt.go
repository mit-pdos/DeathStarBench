package cacheclnt

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"sync/atomic"

	"github.com/harlow/go-micro-services/dialer"
	cached "github.com/harlow/go-micro-services/services/cached/proto"

	"github.com/bradfitz/gomemcache/memcache"
)

var useCached bool

func init() {
	switch os.Getenv("CACHE_TYPE") {
	case "cached":
		useCached = true
	case "memcached":
		useCached = false
	default:
		log.Fatalf("Unknown cache type %v", os.Getenv("CACHE_TYPE"))
	}
}

const (
	CACHE_CLNT_PORT = ":9999"
)

type CacheClnt struct {
	mu  sync.Mutex
	ccs []cached.CachedClient
	ncs int32
}

func MakeCacheClnt() *CacheClnt {
	c := &CacheClnt{
		ccs: make([]cached.CachedClient, 0),
	}

	c.startRPCServer()

	return c
}

func (c *CacheClnt) Get(ctx context.Context, key string) (*memcache.Item, error) {
	if c.ncs == 0 {
		return nil, fmt.Errorf("No caches registered")
	}
	n := c.key2shard(key)
	req := cached.GetRequest{
		Key: key,
	}
	res, err := c.ccs[n].Get(ctx, &req)
	if err != nil {
		log.Fatalf("Error cacheclnt get: %v", err)
	}
	if res.Ok {
		return &memcache.Item{Key: key, Value: res.Val}, nil
	}
	return nil, memcache.ErrCacheMiss
}

func (c *CacheClnt) Set(ctx context.Context, item *memcache.Item) bool {
	n := c.key2shard(item.Key)
	req := cached.SetRequest{
		Key: item.Key,
		Val: item.Value,
	}
	res, err := c.ccs[n].Set(ctx, &req)
	if err != nil {
		log.Fatalf("Error cacheclnt set: %v", err)
	}
	return res.Ok
}

type RegisterCacheRequest struct {
	Addr string
}

type RegisterCacheResponse struct {
	OK bool
}

func (c *CacheClnt) RegisterCache(req *RegisterCacheRequest, rep *RegisterCacheResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Printf("Registering new cache server %v", req.Addr)
	rep.OK = true

	// Make a deep copy of the client slice, so we can atomically swap it with
	// the existing slice. This way, clients don't have to take a lock on the
	// slice when executing RPCs.
	ccs := make([]cached.CachedClient, len(c.ccs))
	for i := range ccs {
		ccs[i] = c.ccs[i]
	}
	// Append the new client.
	ccs = append(ccs, dialClient(req.Addr))
	// Swap in the new slice, which should be done atomically.
	c.ccs = ccs
	// Atomically increase the number by which we mod when selecting a shard.
	atomic.AddInt32(&c.ncs, 1)
	log.Printf("Done registering new cache server %v", req.Addr)
	return nil
}

func (c *CacheClnt) startRPCServer() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", CACHE_CLNT_PORT)
	if err != nil {
		log.Fatalf("Error Listen in Coordinator.registerServer: %v", err)
	}
	go http.Serve(l, nil)
}

func (c *CacheClnt) key2shard(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	shard := int(h.Sum32()) % int(atomic.LoadInt32(&c.ncs))
	return shard
}

func dialClient(addr string) cached.CachedClient {
	// Dial the new server
	// XXX fix
	conn, err := dialer.Dial(addr, nil)
	if err != nil {
		log.Fatalf("Error dial cachesrv: %v", err)
	}
	// Return the new client.
	return cached.NewCachedClient(conn)
}

func UseCached() bool {
	return useCached
}
