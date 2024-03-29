package client

import "sync"

var (
	cacheClientBuildersMutex sync.RWMutex
	cacheClientBuilders      = make(map[string]CacheClientBuilder)
)

// CacheClientBuilder defines builder interface to generate RPCClient.
type CacheClientBuilder interface {
	SetCachedClient(client RPCClient, k, servicePath, serviceMethod string)
	FindCachedClient(k, servicePath, serviceMethod string) RPCClient
	DeleteCachedClient(client RPCClient, k, servicePath, serviceMethod string)
	GenerateClient(k, servicePath, serviceMethod string) (client RPCClient, err error)
}

// RegisterCacheClientBuilder (network string, builder CacheClientBuilder)
func RegisterCacheClientBuilder(network string, builder CacheClientBuilder) {
	cacheClientBuildersMutex.Lock()
	defer cacheClientBuildersMutex.Unlock()

	cacheClientBuilders[network] = builder
}

func getCacheClientBuilder(network string) (CacheClientBuilder, bool) {
	cacheClientBuildersMutex.RLock()
	defer cacheClientBuildersMutex.RUnlock()

	builder, ok := cacheClientBuilders[network]
	return builder, ok
}
