package cache

import (
	pb "cache/cachePB/cachepb"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/cache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	baseURL     string
	basePath    string
	mu          sync.Mutex
	peers       *Map
	httpGetters map[string]*httpGetter
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		baseURL:  self,
		basePath: defaultBasePath,
	}
}

func (hp *HTTPPool) Log(format string, info ...interface{}) {
	log.Printf("[Server %s]: %s", hp.baseURL, fmt.Sprintf(format, info))
}

// ServeHTTP handle all http requests
func (hp *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, hp.basePath) {
		panic("HTTP Pool serving unexpected path: " + r.URL.Path)
	}
	hp.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(hp.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// HTTP Client
type httpGetter struct {
	baseURL string
}

func (client *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		client.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

// Set updates the pool's list of peers.
func (hp *HTTPPool) Set(peers ...string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	hp.peers = NewHash(defaultReplicas, nil)
	hp.peers.Add(peers...)
	hp.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		hp.httpGetters[peer] = &httpGetter{baseURL: peer + hp.basePath}
	}
}

// PickPeer picks a peer according to key
func (hp *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	if peer := hp.peers.Get(key); peer != "" && peer != hp.baseURL {
		hp.Log("Pick peer %s", peer)
		return hp.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
