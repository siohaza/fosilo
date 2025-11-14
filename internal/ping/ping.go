package ping

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
)

type Handler struct {
	conn          *net.UDPConn
	serverInfo    *ServerInfo
	logger        *slog.Logger
	stopChan      chan struct{}
	listenAddress string
}

type ServerInfo struct {
	Name           string `json:"name"`
	PlayersCurrent int    `json:"players_current"`
	PlayersMax     int    `json:"players_max"`
	Map            string `json:"map"`
	GameMode       string `json:"game_mode"`
	GameVersion    string `json:"game_version"`
}

func NewHandler(address string, info *ServerInfo, logger *slog.Logger) *Handler {
	return &Handler{
		serverInfo:    info,
		logger:        logger,
		stopChan:      make(chan struct{}),
		listenAddress: address,
	}
}

func (h *Handler) Start() error {
	addr, err := net.ResolveUDPAddr("udp", h.listenAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	h.conn = conn
	h.logger.Info("ping handler started", "address", h.listenAddress)

	go h.handlePackets()

	return nil
}

func (h *Handler) Stop() {
	close(h.stopChan)
	if h.conn != nil {
		h.conn.Close()
	}
	h.logger.Info("ping handler stopped")
}

func (h *Handler) UpdateServerInfo(info *ServerInfo) {
	h.serverInfo = info
}

func (h *Handler) handlePackets() {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-h.stopChan:
			return
		default:
			n, addr, err := h.conn.ReadFromUDP(buffer)
			if err != nil {
				select {
				case <-h.stopChan:
					return
				default:
					h.logger.Error("failed to read UDP packet", "error", err)
					continue
				}
			}

			if n > 0 {
				h.handlePacket(buffer[:n], addr)
			}
		}
	}
}

func (h *Handler) handlePacket(data []byte, addr *net.UDPAddr) {
	if len(data) == 5 && string(data) == "HELLO" {
		h.handlePing(addr)
	} else if len(data) == 8 && string(data) == "HELLOLAN" {
		h.handleLANPing(addr)
	}
}

func (h *Handler) handlePing(addr *net.UDPAddr) {
	response := []byte("HI")
	_, err := h.conn.WriteToUDP(response, addr)
	if err != nil {
		h.logger.Error("failed to send ping response", "error", err, "addr", addr)
		return
	}
	h.logger.Debug("sent ping response", "addr", addr)
}

func (h *Handler) handleLANPing(addr *net.UDPAddr) {
	jsonData, err := json.Marshal(h.serverInfo)
	if err != nil {
		h.logger.Error("failed to marshal server info", "error", err)
		return
	}

	_, err = h.conn.WriteToUDP(jsonData, addr)
	if err != nil {
		h.logger.Error("failed to send LAN response", "error", err, "addr", addr)
		return
	}
	h.logger.Debug("sent LAN info response", "addr", addr)
}
