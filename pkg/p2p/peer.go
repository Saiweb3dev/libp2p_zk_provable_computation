package p2p

import (
	"context"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	mplex "github.com/libp2p/go-libp2p-mplex"

	host "github.com/libp2p/go-libp2p/core/host"
	ma "github.com/multiformats/go-multiaddr"
)

// NewHost creates a libp2p host with Noise + TCP + Mplex
func NewHost(listenPort int) (host.Host, context.CancelFunc, error) {
	_, cancel := context.WithCancel(context.Background())

	addr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort))

	h, err := libp2p.New(
		libp2p.ListenAddrs(addr),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Transport(tcp.NewTCPTransport),
	)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	fmt.Println("Host created. We are:", h.ID().String())
	for _, a := range h.Addrs() {
		fmt.Println("Listening on:", a.Encapsulate(ma.StringCast(fmt.Sprintf("/p2p/%s", h.ID()))))
	}

	return h, cancel, nil
}
