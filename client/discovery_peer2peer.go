package client

// Peer2PeerDiscovery is a peer-to-peer service discovery.
// It always returns the static server.
type Peer2PeerDiscovery struct {
	server   string
	metadata string
}

// Clone clones this ServiceDiscovery with new servicePath.
func (d *Peer2PeerDiscovery) Clone(_ string) (ServiceDiscovery, error) {
	return d, nil
}

// SetFilter sets the filer.
func (d *Peer2PeerDiscovery) SetFilter(_ ServiceDiscoveryFilter) {}

// GetServices returns the static server
func (d *Peer2PeerDiscovery) GetServices() []*KVPair {
	return []*KVPair{{Key: d.server, Value: d.metadata}}
}

// WatchService returns a nil chan.
func (d *Peer2PeerDiscovery) WatchService() chan []*KVPair {
	return nil
}

func (d *Peer2PeerDiscovery) RemoveWatcher(_ chan []*KVPair) {}

func (d *Peer2PeerDiscovery) Close() {}

// NewPeer2PeerDiscovery returns a new Peer2PeerDiscovery.
func NewPeer2PeerDiscovery(server, metadata string) (*Peer2PeerDiscovery, error) {
	return &Peer2PeerDiscovery{server: server, metadata: metadata}, nil
}
