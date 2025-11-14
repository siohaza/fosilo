package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
)

func (s *Server) sendChatToPlayer(p *player.Player, message string) {
	chatMsg, err := protocol.StringToCP437(message)
	if err != nil {
		s.logger.Error("failed to encode chat message", "error", err)
		return
	}

	packet := protocol.PacketChatMessage{
		PacketID: uint8(protocol.PacketTypeChatMessage),
		PlayerID: p.ID,
		Type:     protocol.ChatTypeSystem,
		Message:  chatMsg,
	}

	s.sendPacket(p, &packet, true)
}

func (s *Server) broadcastChat(message string, chatType protocol.ChatType) {
	chatMsg, err := protocol.StringToCP437(message)
	if err != nil {
		s.logger.Error("failed to encode chat message", "error", err)
		return
	}

	packet := protocol.PacketChatMessage{
		PacketID: uint8(protocol.PacketTypeChatMessage),
		PlayerID: 0,
		Type:     chatType,
		Message:  chatMsg,
	}

	s.broadcastPacket(&packet, true)
}

func (s *Server) KickPlayer(playerID uint8, reason string) {
	p, ok := s.gameState.Players.Get(playerID)
	if !ok {
		return
	}

	if p.SupportsExtension(protocol.ExtensionIDKickReason) {
		chatMsg, err := protocol.StringToCP437(reason)
		if err == nil {
			kickReasonPacket := protocol.PacketChatMessage{
				PacketID: uint8(protocol.PacketTypeChatMessage),
				PlayerID: 255,
				Type:     protocol.ChatTypeSystem,
				Message:  chatMsg,
			}
			s.sendPacket(p, &kickReasonPacket, true)
		}
	} else {
		s.sendChatToPlayer(p, reason)
	}

	s.broadcastChat(fmt.Sprintf("%s was kicked: %s", p.GetName(), reason), protocol.ChatTypeSystem)

	time.AfterFunc(100*time.Millisecond, func() {
		s.network.DisconnectPeerWithReason(p.Peer, false, uint32(protocol.DisconnectReasonKicked))
	})

	s.logger.Info("player kicked", "target", p.GetName(), "reason", reason)
}

func (s *Server) SendChatToAll(message string) {
	s.broadcastChat(message, protocol.ChatTypeSystem)
}

func (s *Server) SendChatToPlayer(p *player.Player, message string) {
	s.sendChatToPlayer(p, message)
}

func (s *Server) SendChatWithType(message string, chatType protocol.ChatType) {
	s.broadcastChat(message, chatType)
}

func (s *Server) RespawnPlayer(playerID uint8) {
	s.respawnPlayer(playerID)
}

func (s *Server) SetPlayerTeam(playerID uint8, team uint8) {
	p, ok := s.gameState.Players.Get(playerID)
	if !ok {
		return
	}
	s.changePlayerTeam(p, team)
}

func (s *Server) BroadcastShortPlayerData(p *player.Player) {
	s.broadcastShortPlayerData(p)
}

func (s *Server) DisconnectPlayerWithReason(p *player.Player, reason uint32) {
	if p == nil || p.Peer == nil {
		return
	}
	s.network.DisconnectPeerWithReason(p.Peer, false, reason)
}

func (s *Server) SendPlayerLeftPacket(playerID uint8) {
	s.broadcastPlayerLeft(playerID)
}

func (s *Server) RestockPlayer(playerID uint8) {
	p, ok := s.gameState.Players.Get(playerID)
	if !ok {
		return
	}

	p.Restock()
	p.LastRestockTime = time.Now()

	s.callbacks.OnRestock(p)
	s.sendRestock(p)
	s.sendPlayerProperties(p)
	s.broadcastShortPlayerData(p)
}

func (s *Server) isCommandDisabled(cmdName string) bool {
	if s.gameState == nil || s.gameState.MapConfig == nil {
		return false
	}

	disabledCommands := s.gameState.MapConfig.Extensions.DisabledCommands
	if len(disabledCommands) == 0 {
		return false
	}

	for _, disabled := range disabledCommands {
		if strings.EqualFold(disabled, cmdName) {
			return true
		}
	}

	return false
}

func (s *Server) handleCommand(p *player.Player, message string) bool {
	if !strings.HasPrefix(message, "/") {
		return false
	}

	parts := strings.Fields(message[1:])
	if len(parts) == 0 {
		return false
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	if s.isCommandDisabled(cmdName) {
		s.sendChatToPlayer(p, fmt.Sprintf("Command '%s' is disabled for this map.", cmdName))
		return true
	}

	if s.luaCommands == nil {
		s.sendChatToPlayer(p, "Commands are not available.")
		return true
	}

	result, err := s.luaCommands.Execute(p, cmdName, args)
	if err != nil {
		if strings.Contains(err.Error(), "unknown command") {
			s.sendChatToPlayer(p, "Unknown command. Type /help for available commands.")
		} else {
			s.sendChatToPlayer(p, fmt.Sprintf("Error: %s", err.Error()))
			s.logger.Error("lua command error", "command", cmdName, "player", p.GetName(), "error", err)
		}
		return true
	}

	if result != "" {
		s.sendChatToPlayer(p, result)
	}

	return true
}

func (s *Server) SaveMap(filename string) (string, error) {
	if s.gameState == nil || s.gameState.Map == nil {
		return "", fmt.Errorf("no map loaded")
	}

	if filename == "" {
		filename = fmt.Sprintf("%s.saved", s.activeMapName)
	}

	if !strings.HasSuffix(filename, ".vxl") {
		filename = filename + ".vxl"
	}

	mapPath := filepath.Join("maps", filename)

	data, err := s.gameState.Map.Write()
	if err != nil {
		return "", fmt.Errorf("failed to serialize map: %w", err)
	}

	err = os.WriteFile(mapPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write map file: %w", err)
	}

	s.logger.Info("map saved", "path", mapPath, "size", len(data))
	return mapPath, nil
}
