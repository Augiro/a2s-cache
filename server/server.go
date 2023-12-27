package server

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Augiro/a2s-cache/util/packets"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net"
)

type challengeMap interface {
	Start(ctx context.Context)
	AddChallenge(key string) Challenge
	Validate(key string, ch Challenge) bool
}

type Cache interface {
	InfoResponse() []byte
	PlayersResponse() []byte
}

type Server struct {
	log         *zap.SugaredLogger
	ip          string
	port        int
	chMapInfo   challengeMap
	chMapPlayer challengeMap
	cache       Cache
}

func New(log *zap.SugaredLogger, ip string, port int, cache Cache) *Server {
	return &Server{
		log,
		ip,
		port,
		NewChallengeMap(log),
		NewChallengeMap(log),
		cache,
	}
}

func (s *Server) Start(ctx context.Context) error {
	pc, err := net.ListenPacket("udp4", fmt.Sprintf("%s:%d", s.ip, s.port))
	if err != nil {
		return fmt.Errorf("unable to listen for UDP packets: %w", err)
	}

	s.log.Info("started UDP server successfully")

	group, innerCTX := errgroup.WithContext(ctx)
	group.Go(func() error { s.chMapInfo.Start(innerCTX); return nil })
	group.Go(func() error { s.chMapPlayer.Start(innerCTX); return nil })
	group.Go(func() error { return s.serve(pc) })

	// Shut down when context closes
	select {
	// Normal shutdown
	case <-ctx.Done():
		s.log.Info("shutting down server...")
		err = pc.Close()
		if err != nil {
			s.log.Errorf("received error shutting down server: %v", err)
		}

		return group.Wait()

	// Something went wrong
	case <-innerCTX.Done():
		return group.Wait()
	}
}

func (s *Server) serve(pc net.PacketConn) error {
	var (
		n    int
		addr net.Addr
		buf  = make([]byte, 1024)
		err  error
	)
	for {
		n, addr, err = pc.ReadFrom(buf)
		if err != nil {
			s.log.Info("unable to read packet on UDP server: %v", err)
			return err
		}

		// Check if A2S_INFO request without challenge
		if n == 25 && bytes.Equal(buf[:n], packets.A2SInfoReq) {
			s.log.Debug("A2S_INFO request received")
			go s.sendChallenge(pc, addr, s.chMapInfo)
			continue
		}

		// Check if A2S_INFO request with challenge
		if n == 29 && bytes.Equal(buf[:25], packets.A2SInfoReq) {
			s.log.Debug("A2S_INFO request with challenge received")
			go s.respond(pc, addr, bytes.Clone(buf[25:n]), s.chMapInfo, s.cache.InfoResponse)
			continue
		}

		// Check if A2S_PLAYER request
		if n == 9 && bytes.Equal(buf[:5], packets.A2SPlayerReq) {
			if bytes.Equal(buf[5:n], packets.A2SPlayerChallengeSuffixTF2) || bytes.Equal(buf[5:n], packets.A2SPlayerChallengeSuffix) {
				s.log.Debug("A2S_PLAYER challenge request received")
				go s.sendChallenge(pc, addr, s.chMapPlayer)
				continue
			}

			s.log.Debug("A2S_PLAYER request received")
			go s.respond(pc, addr, bytes.Clone(buf[5:n]), s.chMapPlayer, s.cache.PlayersResponse)
			continue
		}

		s.log.Debugf("received unknown packet: %x", buf[:n])
	}
}

func (s *Server) sendChallenge(pc net.PacketConn, addr net.Addr, chMap challengeMap) {
	ch := chMap.AddChallenge(addr.String())
	_, err := pc.WriteTo(append(bytes.Clone(packets.ChallengeResp), ch...), addr)
	if err != nil {
		s.log.Errorf("unable to send challenge response: %v", err)
	}
}

func (s *Server) respond(pc net.PacketConn, addr net.Addr, ch Challenge, chMap challengeMap, resp func() []byte) {
	if valid := chMap.Validate(addr.String(), ch); !valid {
		s.log.Debugf("invalid challenge received: %x", ch)
		return
	}

	// If we populated cache yet, drop packet.
	r := resp()
	if r == nil {
		return
	}

	_, err := pc.WriteTo(r, addr)
	if err != nil {
		s.log.Errorf("unable to send response: %v", err)
	}
}
