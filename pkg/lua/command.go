package lua

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/siohaza/fosilo/internal/player"
)

type CommandPermission int

const (
	PermissionNone CommandPermission = iota
	PermissionTrusted
	PermissionGuard
	PermissionModerator
	PermissionAdmin
	PermissionManager
)

type LuaCommand struct {
	Name        string
	Aliases     []string
	Permission  CommandPermission
	Description string
	Usage       string
	Handler     string
	VM          *VM
}

type CommandManager struct {
	commands map[string]*LuaCommand
	aliases  map[string]string
	logger   *slog.Logger
}

func NewCommandManager(logger *slog.Logger) *CommandManager {
	return &CommandManager{
		commands: make(map[string]*LuaCommand),
		aliases:  make(map[string]string),
		logger:   logger,
	}
}

func (cm *CommandManager) LoadCommands(commandsDir string, api *GameAPI) error {
	files, err := os.ReadDir(commandsDir)
	if err != nil {
		return fmt.Errorf("failed to read commands directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".lua") {
			continue
		}

		commandPath := filepath.Join(commandsDir, file.Name())
		if err := cm.LoadCommandFile(commandPath, api); err != nil {
			cm.logger.Warn("failed to load command file", "file", file.Name(), "error", err)
			continue
		}
	}

	cm.logger.Info("loaded lua commands", "count", len(cm.commands))
	return nil
}

func (cm *CommandManager) Reload(commandsDir string, api *GameAPI) error {
	cm.commands = make(map[string]*LuaCommand)
	cm.aliases = make(map[string]string)

	return cm.LoadCommands(commandsDir, api)
}

func (cm *CommandManager) LoadCommandFile(path string, api *GameAPI) error {
	vm := NewVM()

	if api != nil {
		api.RegisterFunctions(vm)
	}

	if err := vm.LoadFile(path); err != nil {
		return err
	}

	name, err := vm.GetGlobalString("name")
	if err != nil {
		return fmt.Errorf("command missing 'name': %w", err)
	}

	cmd := &LuaCommand{
		Name: name,
		VM:   vm,
	}

	if aliases, err := vm.GetGlobalString("aliases"); err == nil {
		cmd.Aliases = strings.Split(aliases, ",")
		for i, alias := range cmd.Aliases {
			cmd.Aliases[i] = strings.TrimSpace(alias)
		}
	}

	if desc, err := vm.GetGlobalString("description"); err == nil {
		cmd.Description = desc
	}

	if usage, err := vm.GetGlobalString("usage"); err == nil {
		cmd.Usage = usage
	}

	if perm, err := vm.GetGlobalString("permission"); err == nil {
		cmd.Permission = parsePermission(perm)
	}

	if handler, err := vm.GetGlobalString("handler"); err == nil {
		cmd.Handler = handler
	} else {
		cmd.Handler = "execute"
	}

	cm.Register(cmd)
	return nil
}

func (cm *CommandManager) Register(cmd *LuaCommand) {
	cm.commands[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		if alias != "" {
			cm.aliases[alias] = cmd.Name
		}
	}
}

func (cm *CommandManager) Get(name string) *LuaCommand {
	name = strings.ToLower(name)
	if canonical, ok := cm.aliases[name]; ok {
		return cm.commands[canonical]
	}
	return cm.commands[name]
}

func (cm *CommandManager) Execute(p *player.Player, cmdName string, args []string) (string, error) {
	cmd := cm.Get(cmdName)
	if cmd == nil {
		return "", fmt.Errorf("unknown command: %s", cmdName)
	}

	if !hasPermission(p, cmd.Permission) {
		return "", fmt.Errorf("you don't have permission to use this command")
	}

	state := cmd.VM.State()
	state.Global(cmd.Handler)
	if !state.IsFunction(-1) {
		state.Pop(1)
		return "", fmt.Errorf("command handler not found: %s", cmd.Handler)
	}

	PushPlayer(state, p)

	state.NewTable()
	state.PushString(cmdName)
	state.RawSetInt(-2, 0)
	for i, arg := range args {
		state.PushString(arg)
		state.RawSetInt(-2, i+1)
	}

	if err := state.ProtectedCall(2, 1, 0); err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	result := ""
	if state.IsString(-1) {
		result, _ = state.ToString(-1)
	}
	state.Pop(1)

	return result, nil
}

func (cm *CommandManager) List(p *player.Player) []*LuaCommand {
	var commands []*LuaCommand
	for _, cmd := range cm.commands {
		if hasPermission(p, cmd.Permission) {
			commands = append(commands, cmd)
		}
	}
	return commands
}

func parsePermission(perm string) CommandPermission {
	switch strings.ToLower(perm) {
	case "trusted":
		return PermissionTrusted
	case "guard":
		return PermissionGuard
	case "moderator", "mod":
		return PermissionModerator
	case "admin":
		return PermissionAdmin
	case "manager":
		return PermissionManager
	default:
		return PermissionNone
	}
}

func hasPermission(p *player.Player, required CommandPermission) bool {
	if required == PermissionNone {
		return true
	}

	playerPerm := getPlayerPermission(p)
	return playerPerm >= required
}

func getPlayerPermission(p *player.Player) CommandPermission {
	perms := p.Permissions

	if perms&(1<<5) != 0 {
		return PermissionManager
	}
	if perms&(1<<4) != 0 {
		return PermissionAdmin
	}
	if perms&(1<<3) != 0 {
		return PermissionModerator
	}
	if perms&(1<<2) != 0 {
		return PermissionGuard
	}
	if perms&(1<<1) != 0 {
		return PermissionTrusted
	}

	return PermissionNone
}
