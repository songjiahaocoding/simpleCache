package cache

import pb "cache/cachePB/cachepb"

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a client.
// It is for the HTTP client to get a result from a remote peer
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
