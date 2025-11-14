package masterserver

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/codecat/go-enet"
)

type Client struct {
	host       enet.Host
	peer       enet.Peer
	domain     string
	domainPort uint16
	serverName string
	gameMode   string
	mapName    string
	maxPlayers uint8
	port       uint16
	enabled    bool
	connected  bool
	logger     *slog.Logger
}

func New(domain string, domainPort int, serverName string, gameMode string, mapName string, port int, maxPlayers int, logger *slog.Logger) (*Client, error) {
	host, err := enet.NewHost(nil, 1, 1, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	if err := host.CompressWithRangeCoder(); err != nil {
		return nil, err
	}

	return &Client{
		host:       host,
		domain:     domain,
		domainPort: uint16(domainPort),
		serverName: serverName,
		gameMode:   gameMode,
		mapName:    mapName,
		maxPlayers: uint8(maxPlayers),
		port:       uint16(port),
		enabled:    false,
		connected:  false,
		logger:     logger,
	}, nil
}

func (c *Client) Enable() {
	c.enabled = true
	c.logger.Info("master server enabled", "domain", c.domain)
}

func (c *Client) Disable() {
	c.enabled = false
	c.logger.Info("master server disabled")
}

func (c *Client) UpdateMap(mapName string) {
	c.mapName = mapName
	if c.enabled && c.connected {
		c.sendMajorUpdate()
	}
}

func (c *Client) UpdatePlayerCount(count uint8) {
	if c.enabled && c.connected {
		c.sendPlayerCountUpdate(count)
	}
}

func (c *Client) sendMajorUpdate() {
	var buf bytes.Buffer

	buf.WriteByte(c.maxPlayers)
	binary.Write(&buf, binary.LittleEndian, c.port)
	buf.WriteString(c.serverName)
	buf.WriteByte(0)
	buf.WriteString(c.gameMode)
	buf.WriteByte(0)
	buf.WriteString(c.mapName)
	buf.WriteByte(0)

	packet, err := enet.NewPacket(buf.Bytes(), enet.PacketFlagReliable)
	if err != nil {
		c.logger.Error("failed to create major update packet", "error", err)
		return
	}

	if err := c.peer.SendPacket(packet, 0); err != nil {
		c.logger.Error("failed to send major update", "error", err)
	}
}

func (c *Client) sendPlayerCountUpdate(count uint8) {
	packet, err := enet.NewPacket([]byte{count}, enet.PacketFlagReliable)
	if err != nil {
		c.logger.Error("failed to create player count packet", "error", err)
		return
	}

	if err := c.peer.SendPacket(packet, 0); err != nil {
		c.logger.Error("failed to send player count update", "error", err)
	}
}

func (c *Client) Service() {
	if c.enabled {
		if c.peer == nil {
			address := enet.NewAddress(c.domain, c.domainPort)

			peer, err := c.host.Connect(address, 1, 31)
			if err != nil {
				c.logger.Error("failed to connect to master server", "error", err)
				return
			}
			c.peer = peer
			c.connected = false
		}

		event := c.host.Service(0)
		if event.GetType() != enet.EventNone {
			switch event.GetType() {
			case enet.EventConnect:
				c.logger.Info("connected to master server")
				c.connected = true
				c.sendMajorUpdate()

			case enet.EventDisconnect:
				c.logger.Warn("disconnected from master server")
				c.connected = false
				c.peer = nil

			case enet.EventReceive:
				event.GetPacket().Destroy()
			}
		}
	} else if c.peer != nil && c.connected {
		c.peer.DisconnectLater(0)
		c.host.Service(0)
		c.connected = false
		c.peer = nil
	}
}

func (c *Client) Start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			<-ticker.C
			c.Service()
		}
	}()
}

func (c *Client) Destroy() {
	if c.peer != nil && c.connected {
		c.peer.Disconnect(0)
		time.Sleep(100 * time.Millisecond)
		c.host.Service(0)
	}
}
