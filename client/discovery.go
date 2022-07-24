package client

// ServiceDiscovery defines ServiceDiscovery of zookeeper, etcd and consul
type (
	ServiceDiscovery interface {
		GetServices() []*KVPair
		WatchService() chan []*KVPair
		RemoveWatcher(ch chan []*KVPair)
		Clone(servicePath string) (ServiceDiscovery, error)
		SetFilter(ServiceDiscoveryFilter)
		Close()
	}

	// KVPair contains a key and a string.
	KVPair struct {
		Key   string
		Value string
	}

	// ServiceDiscoveryFilter can be used to filter services with customized logics.
	// Servers can register its services but clients can use the customized filter to select some services.
	// It returns true if ServiceDiscovery wants to use this service, otherwise it returns false.
	ServiceDiscoveryFilter func(kvp *KVPair) bool
)
