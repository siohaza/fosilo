package network

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/codecat/go-enet"
)

type Server struct {
	host     enet.Host
	port     uint16
	maxPeers int
	logger   *slog.Logger
}

type Event struct {
	Type      EventType
	Peer      enet.Peer
	Data      []byte
	ChannelID uint8
}

type EventType int

const (
	EventTypeNone EventType = iota
	EventTypeConnect
	EventTypeDisconnect
	EventTypeReceive
)

func NewServer(port int, maxPeers int, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		port:     uint16(port),
		maxPeers: maxPeers,
		logger:   logger,
	}, nil
}

func (s *Server) Start() error {
	address := enet.NewListenAddress(s.port)

	var err error
	s.host, err = enet.NewHost(address, uint64(s.maxPeers), 1, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to create ENet host: %w", err)
	}

	if err := s.host.CompressWithRangeCoder(); err != nil {
		return fmt.Errorf("failed to setup range coder compression: %w", err)
	}

	s.logger.Info("server started", "port", s.port, "max_peers", s.maxPeers)
	return nil
}

func (s *Server) Stop() {
	if s.host != nil {
		s.host.Destroy()
		s.logger.Info("server stopped")
	}
}

func (s *Server) Service(timeout time.Duration) (*Event, error) {
	if s.host == nil {
		return nil, fmt.Errorf("server not started")
	}

	timeoutMs := uint32(timeout.Milliseconds())
	enetEvent := s.host.Service(timeoutMs)

	if enetEvent == nil {
		return &Event{Type: EventTypeNone}, nil
	}

	event := &Event{
		Peer: enetEvent.GetPeer(),
	}

	switch enetEvent.GetType() {
	case enet.EventConnect:
		event.Type = EventTypeConnect
		s.logger.Debug("peer connected", "peer", enetEvent.GetPeer().GetAddress())

	case enet.EventDisconnect:
		event.Type = EventTypeDisconnect
		s.logger.Debug("peer disconnected", "peer", enetEvent.GetPeer().GetAddress())

	case enet.EventReceive:
		event.Type = EventTypeReceive
		packet := enetEvent.GetPacket()
		if packet != nil {
			event.Data = packet.GetData()
			event.ChannelID = enetEvent.GetChannelID()
			packet.Destroy()
		}
	}

	return event, nil
}

func (s *Server) SendPacket(peer enet.Peer, data []byte, reliable bool) error {
	if peer == nil {
		return fmt.Errorf("peer is nil")
	}

	flags := enet.PacketFlagUnsequenced
	if reliable {
		flags = enet.PacketFlagReliable
	}

	packet, err := enet.NewPacket(data, flags)
	if err != nil {
		return fmt.Errorf("failed to create packet: %w", err)
	}

	if err := peer.SendPacket(packet, 0); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}

func (s *Server) Broadcast(data []byte, reliable bool) error {
	if s.host == nil {
		return fmt.Errorf("server not started")
	}

	flags := enet.PacketFlagUnsequenced
	if reliable {
		flags = enet.PacketFlagReliable
	}

	if err := s.host.BroadcastBytes(data, 0, flags); err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	return nil
}

func (s *Server) DisconnectPeer(peer enet.Peer, immediate bool) {
	s.DisconnectPeerWithReason(peer, immediate, 0)
}

func (s *Server) DisconnectPeerWithReason(peer enet.Peer, immediate bool, reason uint32) {
	if peer == nil {
		return
	}

	if immediate {
		peer.DisconnectNow(reason)
	} else {
		peer.Disconnect(reason)
	}
}

func (s *Server) GetPeerCount() int {
	return 0
}
