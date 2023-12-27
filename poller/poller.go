package poller

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Augiro/a2s-cache/util/packets"
	"go.uber.org/zap"
	"net"
	"time"
)

const PollTimeout = 3 * time.Second

type Cache interface {
	SetInfoResponse(resp []byte)
	SetPlayersResponse(resp []byte)
}

type Poller struct {
	log   *zap.SugaredLogger
	ip    string
	port  int
	cache Cache
}

func New(log *zap.SugaredLogger, ip string, port int, cache Cache) *Poller {
	return &Poller{
		log, ip, port, cache,
	}
}

func (p *Poller) Start(ctx context.Context) {
	// Do one initial poll straight away.
	err := p.poll()
	if err != nil {
		p.log.Errorf("poll failed: %v", err)
	}

	ticker := time.NewTicker(10 * time.Second)
	p.log.Info("started poller successfully")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err = p.poll()
			if err != nil {
				p.log.Errorf("poll failed: %v", err)
			}
		}
	}
}

func (p *Poller) poll() error {
	err := p.execQuery("A2S_INFO", packets.A2SInfoReq, false, p.cache.SetInfoResponse)
	if err != nil {
		return fmt.Errorf("A2S_INFO poll failed: %w", err)
	}
	err = p.execQuery("A2S_PLAYER", packets.A2SPlayerReq, true, p.cache.SetPlayersResponse)
	if err != nil {
		return fmt.Errorf("A2S_PLAYER poll failed: %w", err)
	}
	return nil
}

// execQuery executes a server query based on the req parameter, and passes the response to the store function.
// If initiaCH is true, we will append 0xffffffff to req, as is necessary for some queries.
func (p *Poller) execQuery(name string, req []byte, initialCH bool, store func([]byte)) error {
	p.log.Debugf("polling %s from server...", name)

	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", p.ip, p.port))
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("unable to dial server over UDP: %w", err)
	}

	closed := make(chan bool)
	defer func() {
		closed <- true
		err = conn.Close()
		if err != nil {
			p.log.Errorf("unable to close UDP connection to server: %v", err)
		}
	}()

	// Timeout in case we don't get any response
	go func() {
		select {
		case <-closed:
			return
		case <-time.After(PollTimeout):
			p.log.Errorf("%s poll timed out", name)
			conn.Close()
		}
	}()

	// A2S_PLAYER needs to send 0xffffffff as challenge, if we want to get a challenge response
	// (gotta love inconsistent APIs).
	initialReq := req
	if initialCH {
		initialReq = append(bytes.Clone(req), 0xff, 0xff, 0xff, 0xff)
	}
	_, err = conn.Write(initialReq)
	if err != nil {
		return fmt.Errorf("unable to write to server over UDP: %w", err)
	}

	buf := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("unable to read server over UDP: %w", err)
	}

	if !bytes.Equal(buf[:5], packets.ChallengeResp) {
		return fmt.Errorf("did not get ChallengeResp from server: %w", err)
	}

	_, err = conn.Write(append(req, buf[5:n]...))
	if err != nil {
		return fmt.Errorf("unable to write to server over UDP: %w", err)
	}

	n, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("unable to write to server over UDP: %w", err)
	}

	store(bytes.Clone(buf[:n]))
	p.log.Debugf("successfully polled %s", name)
	return nil
}
