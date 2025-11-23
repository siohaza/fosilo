# Lua API documentation

Documentation about internals of server for server side scripting for both gamemodes and commands.

If you have no idea how to work with Lua, here is a couple of resources which you can use to get to know the language:
- [Lua 5.2 reference manual](https://www.lua.org/manual/5.2/manual.html)
- [Learn X in Y minutes - Lua](https://learnxinyminutes.com/docs/lua)

## Table of Contents

1. [Player Table Structure](#player-table-structure)
2. [Map Functions](#map-functions)
3. [Player Functions](#player-functions)
4. [Player Modification Functions](#player-modification-functions)
5. [Team and Score Functions](#team-and-score-functions)
6. [Chat and Communication Functions](#chat-and-communication-functions)
7. [Network Packet Functions](#network-packet-functions)
8. [Admin and Moderation Functions](#admin-and-moderation-functions)
9. [Vote Functions](#vote-functions)
10. [Utility Functions](#utility-functions)
11. [Timer Functions](#timer-functions)
12. [System Functions](#system-functions)
13. [Gamemode System](#gamemode-system)
14. [Command System](#command-system)
15. [Constants and Enums](#constants-and-enums)

## Player Table Structure

When you receive a player object from functions like `get_player()` or in event hooks, it has the following structure:

```lua
player = {
    id = number,           -- Player ID (0-31)
    name = string,         -- Player name
    team = number,         -- Team number (0 or 1)
    alive = boolean,       -- Is player alive
    hp = number,           -- Health points (0-100)
    kills = number,        -- Kill count
    deaths = number,       -- Death count
    has_intel = boolean,   -- Is carrying intel
    permissions = number,  -- Permission bitmask
    position = {           -- Position table
        [1] = x,          -- X coordinate
        [2] = y,          -- Y coordinate
        [3] = z           -- Z coordinate
    }
}
```

## Map Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `find_top_block(x, y)` | `x` (number): X coordinate<br>`y` (number): Y coordinate | `number`: Z coordinate of top block, or -1 if none found | Finds the topmost solid block at the given X,Y coordinates |
| `get_block(x, y, z)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate | `number`: Block color as uint32, or 0 if air | Gets the color value of a block at the specified position |
| `is_solid(x, y, z)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate | `boolean`: True if block is solid | Checks if a block at the specified position is solid |
| `set_block(x, y, z, color)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate<br>`color` (number): Block color as uint32 | None | Sets a block at the specified position with the given color |
| `destroy_block(x, y, z)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate | None | Removes a block at the specified position |
| `get_map_width()` | None | `number`: Map width | Gets the map width |
| `get_map_height()` | None | `number`: Map height | Gets the map height |
| `get_map_depth()` | None | `number`: Map depth (usually 64) | Gets the map depth |
| `get_map_name()` | None | `string`: Current map name | Gets the current map name |
| `is_valid_position(x, y, z)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate | `boolean`: True if position is valid | Checks if a position is within map bounds |

### Example: Map Functions

```lua
local top_z = find_top_block(256, 256)
if top_z >= 0 then
    print("Top block at center is at Z=" .. top_z)
end

if is_solid(x, y, z) then
    print("Block is solid")
end

local red = rgb_to_color(255, 0, 0)
set_block(256, 256, 32, red)
```

## Player Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `get_player(id)` | `id` (number): Player ID (0-31) | `table`: Player table, or nil if not found | Gets a player by their ID |
| `get_player_by_id(id)` | `id` (number): Player ID (0-31) | `table`: Player table, or nil if not found | Alias for get_player() |
| `get_player_count()` | None | `number`: Number of connected players | Gets the number of connected players |
| `get_player_by_name(name)` | `name` (string): Player name | `table`: Player table, or nil if not found | Finds a player by their name (exact match) |
| `get_player_position(id)` | `id` (number): Player ID | `number, number, number`: x, y, z coordinates | Gets a player's position |
| `get_player_team(id)` | `id` (number): Player ID | `number`: Team number (0 or 1), or -1 if not found | Gets a player's team |
| `get_player_name(id)` | `id` (number): Player ID | `string`: Player name, or empty string if not found | Gets a player's name |
| `get_player_weapon(id)` | `id` (number): Player ID | `number`: Weapon type (0=Rifle, 1=SMG, 2=Shotgun), or -1 if not found | Gets a player's weapon |
| `get_player_ip(id)` | `id` (number): Player ID | `string`: Player's IP address | Gets a player's IP address |
| `is_player_alive(id)` | `id` (number): Player ID | `boolean`: True if player is alive | Checks if a player is alive |
| `get_player_ping(id)` | `id` (number): Player ID | `number`: Ping in milliseconds, or -1 if unavailable | Gets the player's current network latency |
| `get_player_ammo(id)` | `id` (number): Player ID | `number, number`: Magazine ammo, reserve ammo | Gets the player's current ammunition |
| `get_player_grenades(id)` | `id` (number): Player ID | `number`: Number of grenades | Gets a player's grenade count |
| `get_player_blocks(id)` | `id` (number): Player ID | `number`: Number of blocks | Gets a player's block count |
| `get_player_color(id)` | `id` (number): Player ID | `number, number, number`: R, G, B values (0-255) | Gets the player's color |
| `get_player_tool(id)` | `id` (number): Player ID | `number`: Tool type (0=Spade, 1=Block, 2=Gun, 3=Grenade), or -1 if not found | Gets the player's currently equipped tool |
| `get_player_state(id)` | `id` (number): Player ID | `table`: State table with fields: `crouching`, `sprinting`, `airborne` | Gets the player's current movement state |
| `get_player_orientation(id)` | `id` (number): Player ID | `number, number, number`: Orientation vector (x, y, z) | Gets the direction the player is looking |

### Example: Player Functions

```lua
local player = get_player(5)
if player then
    print(player.name .. " has " .. player.hp .. " HP")
end

local x, y, z = get_player_position(player_id)
print("Player is at " .. x .. ", " .. y .. ", " .. z)

local ping = get_player_ping(player.id)
if ping >= 0 then
    print("Ping: " .. ping .. "ms")
end

local mag, reserve = get_player_ammo(player.id)
print("Ammo: " .. mag .. "/" .. reserve)

local r, g, b = get_player_color(player.id)
print("Player color: RGB(" .. r .. ", " .. g .. ", " .. b .. ")")

local state = get_player_state(player.id)
if state then
    if state.airborne then
        print("Player is in the air!")
    end
    if state.sprinting then
        print("Player is sprinting!")
    end
end
```

## Player Modification Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `set_player_position(id, x, y, z)` | `id` (number): Player ID<br>`x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate | None | Teleports a player to the specified position. Automatically resets velocity to prevent fall damage |
| `set_player_hp(id, hp)` | `id` (number): Player ID<br>`hp` (number): Health points (0-100) | None | Sets a player's health. Setting HP to 0 will kill the player |
| `heal_player(id)` | `id` (number): Player ID | None | Fully restocks a player (restores HP to 100, refills ammo, grenades, and blocks) |
| `set_player_team(id, team)` | `id` (number): Player ID<br>`team` (number): Team number (0 or 1) | None | Changes a player's team |
| `set_player_weapon(id, weapon)` | `id` (number): Player ID<br>`weapon` (number): Weapon type (0=Rifle, 1=SMG, 2=Shotgun) | None | Changes a player's weapon |
| `set_player_ammo(id, magazine, reserve)` | `id` (number): Player ID<br>`magazine` (number): Magazine ammo<br>`reserve` (number): Reserve ammo | None | Sets a player's ammunition |
| `set_player_grenades(id, count)` | `id` (number): Player ID<br>`count` (number): Number of grenades (0-3) | None | Sets a player's grenade count |
| `set_player_blocks(id, count)` | `id` (number): Player ID<br>`count` (number): Number of blocks (0-50) | None | Sets a player's block count |
| `set_player_permission(id, permission)` | `id` (number): Player ID<br>`permission` (string): Permission level ("trusted", "guard", "moderator", "admin", "manager") | None | Sets a player's permission level |
| `set_player_orientation(id, x, y, z)` | `id` (number): Player ID<br>`x` (number): Orientation X<br>`y` (number): Orientation Y<br>`z` (number): Orientation Z | None | Sets the direction the player is looking |
| `respawn_player(id)` | `id` (number): Player ID | None | Respawns a dead player. This sets the player alive and restores HP but does NOT send spawn packet or teleport. Use with `set_player_position()` |
| `kill_player(id)` | `id` (number): Player ID | None | Kills a player |

### Example: Player Modification Functions

```lua
set_player_position(player.id, 256, 256, 32)
set_player_permission(player.id, "admin")
```

## Team and Score Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `get_team_score(team)` | `team` (number): Team number (0 or 1) | `number`: Team score | Gets a team's score |
| `set_team_score(team, score)` | `team` (number): Team number (0 or 1)<br>`score` (number): New score | None | Sets a team's score |
| `set_intel_position(x, y, z, team)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate<br>`team` (number): Team number (0 or 1) | None | Sets the intel spawn position for a team |
| `set_base_position(x, y, z, team)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate<br>`team` (number): Team number (0 or 1) | None | Sets the base position for a team (used for respawn) |
| `get_intel_position(team)` | `team` (number): Team number (0 or 1) | `number, number, number`: Intel position (x, y, z) | Gets the intel position for a team |
| `get_base_position(team)` | `team` (number): Team number (0 or 1) | `number, number, number`: Base position (x, y, z) | Gets the base position for a team |
| `get_spawn_location(team)` | `team` (number): Team number (0 or 1) | `number, number, number`: Spawn position (x, y, z) | Gets a spawn location for the specified team |

### Example: Team and Score Functions

```lua
local score = get_team_score(0)
set_team_score(0, score + 1)
```

## Chat and Communication Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `send_chat(player_id, message)` | `player_id` (number): Target player ID<br>`message` (string): Message to send | None | Sends a chat message to a specific player |
| `broadcast_chat(message)` | `message` (string): Message to broadcast | None | Sends a chat message to all players |
| `send_big_message(message)` | `message` (string): Message to display | None | Sends a large center-screen message to all players |
| `send_info_message(message)` | `message` (string): Info message | None | Sends a blue info message to all players |
| `send_warning_message(message)` | `message` (string): Warning message | None | Sends a yellow warning message to all players |
| `send_error_message(message)` | `message` (string): Error message | None | Sends a red error message to all players |

### Example: Chat and Communication Functions

```lua
broadcast_chat("Round starting in 5 seconds!")
```

## Network Packet Functions

These functions send network packets to clients without updating server side state.

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `send_player_position_packet(player_id, x, y, z, ox, oy, oz, to_player_id)` | `player_id` (number): Player ID to show position for<br>`x, y, z` (number): Position coordinates<br>`ox, oy, oz` (number): Orientation vector<br>`to_player_id` (number): Target player ID (255 to broadcast to all) | None | Sends a position/orientation packet without updating server state. Useful for visual-only position updates |
| `send_intel_position_packet(object_id, team, x, y, z)` | `object_id` (number): Intel/object ID (0-1)<br>`team` (number): Team number (0 or 1)<br>`x, y, z` (number): Position coordinates | None | Sends a MoveObject packet without updating intel state. Perfect for using intel packets as HP bars or custom indicators |
| `send_territory_capture(player_id, entity_id, winning, state)` | `player_id` (number): Player ID<br>`entity_id` (number): Territory entity ID<br>`winning` (number): Winning team<br>`state` (number): Territory state | None | Sends a territory capture packet. Already packet-only, doesn't update state |
| `send_progress_bar(entity_id, capturing_team, rate, progress)` | `entity_id` (number): Entity ID<br>`capturing_team` (number): Capturing team<br>`rate` (number): Capture rate (-128 to 127)<br>`progress` (number): Progress value (0.0 to 1.0) | None | Sends a progress bar packet. Perfect for custom progress indicators |

### Example: Network Packet Functions

```lua
-- Display Team 0's HP as a floating intel object
function update_hp_display()
    local team0_hp = calculate_team_hp(0)
    local hp_x = 256
    local hp_y = 256
    local hp_z = 10 + (team0_hp / 10)  -- Height based on HP

    -- Shows intel at position without actually moving the real intel
    send_intel_position_packet(0, 0, hp_x, hp_y, hp_z)
end

-- Show painting progress using a progress bar
function show_paint_progress(player, painted_blocks, total_blocks)
    local progress = painted_blocks / total_blocks
    local rate = 10  -- Positive rate shows as "capturing"

    send_progress_bar(player.id, player.team, rate, progress)
end

-- Send a visual position update to specific player
function show_ghost_position(ghost_player_id, x, y, z, viewer_id)
    local ox, oy, oz = 0, 0, 0  -- Default orientation

    -- Only the viewer sees the ghost, doesn't actually move the player
    send_player_position_packet(ghost_player_id, x, y, z, ox, oy, oz, viewer_id)
end

-- Broadcast visual effect to all players
function show_flag_carrier_beacon(carrier_id)
    local x, y, z = get_player_position(carrier_id)
    z = z - 5  -- Show beacon above carrier

    -- Broadcast to all players (255 = broadcast)
    send_intel_position_packet(1, carrier_id.team, x, y, z)
end
```

## Admin and Moderation Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `ban_player(ip, name, reason, banned_by, duration_hours)` | `ip` (string): IP address to ban<br>`name` (string): Player name<br>`reason` (string): Ban reason<br>`banned_by` (string): Name of person issuing ban<br>`duration_hours` (number): Ban duration in hours (0 for permanent) | `boolean, string`: Success status, error message | Bans a player by IP address |
| `unban_ip(ip)` | `ip` (string): IP address to unban | `boolean, string`: Success status, error message | Unbans an IP address |
| `is_banned(ip)` | `ip` (string): IP address to check | `boolean`: True if banned | Checks if an IP address is banned |
| `kick_player_cmd(id, reason)` | `id` (number): Player ID<br>`reason` (string): Kick reason | `boolean, string`: Success status, error message | Kicks a player from the server |
| `disconnect_player(id, reason_code)` | `id` (number): Player ID<br>`reason_code` (number): Disconnect reason code | `boolean, string`: Success status, error message | Disconnects a player with a specific disconnect reason code |
| `has_permission(player_id, permission)` | `player_id` (number): Player ID<br>`permission` (string): Permission level to check ("trusted", "guard", "moderator", "admin", "manager") | `boolean`: True if player has permission | Checks if a player has a specific permission level or higher |

### Example: Admin and Moderation Functions

```lua
local success, err = ban_player("127.0.0.1", "Cheater", "Hacking", "Admin", 24)
if not success then
    print("Ban failed: " .. err)
end

if has_permission(player.id, "admin") then
    print("Player is an admin")
end
```

## Vote Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `start_votekick(instigator_id, victim_id, reason)` | `instigator_id` (number): ID of player starting vote<br>`victim_id` (number): ID of player to kick<br>`reason` (string): Reason for kick | `boolean, string`: Success status, error message | Starts a votekick |
| `start_votemap(instigator_id)` | `instigator_id` (number): ID of player starting vote | `boolean, string`: Success status, error message | Starts a map vote |
| `cast_vote(player_id, choice)` | `player_id` (number): Player ID<br>`choice` (boolean\|string\|number): Vote choice (boolean for kick, string/number for map) | `boolean, string`: Success status, error message | Casts a vote in the current poll |
| `cancel_vote(player_id)` | `player_id` (number): Player ID | `boolean, string`: Success status, error message | Cancels the current vote (instigator or admin only) |
| `has_active_vote()` | None | `boolean`: True if a vote is currently active | Checks if a vote is currently active |
| `get_vote_choices()` | None | `table\|nil`: Array of map choices, or nil if not a map vote | Gets the available vote choices |
| `get_vote_type()` | None | `string`: "kick" or "map", or nil if no active vote | Gets the type of the current vote |

## Utility Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `distance_3d(x1, y1, z1, x2, y2, z2)` | `x1, y1, z1` (number): First point coordinates<br>`x2, y2, z2` (number): Second point coordinates | `number`: Distance | Calculates 3D Euclidean distance between two points |
| `distance_2d(x1, y1, x2, y2)` | `x1, y1` (number): First point coordinates<br>`x2, y2` (number): Second point coordinates | `number`: Distance | Calculates 2D distance between two points (ignores Z) |
| `rgb_to_color(r, g, b)` | `r` (number): Red (0-255)<br>`g` (number): Green (0-255)<br>`b` (number): Blue (0-255) | `number`: Color as uint32 | Converts RGB values to a uint32 color |
| `color_to_rgb(color)` | `color` (number): Color as uint32 | `number, number, number`: R, G, B values | Converts a uint32 color to RGB values |
| `for_each_player(callback)` | `callback` (function): Function to call for each player, receives player table | None | Iterates over all connected players and calls a callback function for each |
| `clamp(value, min, max)` | `value` (number): Value to clamp<br>`min` (number): Minimum value<br>`max` (number): Maximum value | `number`: Clamped value | Clamps a value between min and max |
| `lerp(a, b, t)` | `a` (number): Start value<br>`b` (number): End value<br>`t` (number): Interpolation factor (0-1) | `number`: Interpolated value | Linear interpolation between two values |

### Example: Utility Functions

```lua
local px, py, pz = get_player_position(player.id)
local ix, iy, iz = get_intel_position(0)
local dist = distance_3d(px, py, pz, ix, iy, iz)
if dist < 5 then
    print("Player is near intel!")
end

local red = rgb_to_color(255, 0, 0)
local green = rgb_to_color(0, 255, 0)
local blue = rgb_to_color(0, 0, 255)

local r, g, b = color_to_rgb(block_color)

for_each_player(function(player)
    if player.alive and player.hp < 50 then
        send_chat(player.id, "You're low on health!")
    end
end)

local safe_damage = clamp(damage, 0, 100)
local half_way = lerp(0, 100, 0.5)  -- Returns 50
```

## Timer Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `schedule_callback(seconds, callback_name, repeat)` | `seconds` (number): Delay in seconds<br>`callback_name` (string): Name of function to call<br>`repeat` (boolean): If true, repeats indefinitely | `number`: Timer ID | Schedules a callback function to be called after a delay |
| `cancel_callback(timer_id)` | `timer_id` (number): Timer ID returned from schedule_callback | None | Cancels a scheduled callback |
| `get_timer_info(timer_id)` | `timer_id` (number): Timer ID | `table\|nil`: Timer info with fields: `remaining`, `interval`, `repeat` | Gets information about a scheduled timer |
| `pause_timer(timer_id)` | `timer_id` (number): Timer ID | `boolean, string`: Success status, error message | Not yet implemented |
| `resume_timer(timer_id)` | `timer_id` (number): Timer ID | `boolean, string`: Success status, error message | Not yet implemented |

### Example: Timer Functions

```lua
function round_timer()
    broadcast_chat("Round ended!")
end

-- Call round_timer() after 180 seconds, once
local timer_id = schedule_callback(180, "round_timer", false)

function tick_sound()
    send_big_message("TICK")
end

-- Call tick_sound() every second
local tick_timer = schedule_callback(1, "tick_sound", true)

-- Cancel a timer
cancel_callback(timer_id)

-- Get timer info
local info = get_timer_info(timer_id)
if info then
    print("Timer has " .. info.remaining .. " seconds remaining")
end
```

## System Functions

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `reload_commands()` | None | `boolean, string`: Success status, error message | Reloads all Lua commands without restarting the server |
| `reload_gamemode()` | None | `boolean, string`: Success status, error message | Reloads the current gamemode without restarting the server |
| `get_available_commands(player_id)` | `player_id` (number): Player ID | `table`: Array of command tables with fields: `name`, `description`, `usage`, `aliases` | Gets all commands available to a player based on their permissions |
| `get_config_password(role)` | `role` (string): Role name ("trusted", "guard", "moderator", "admin", "manager") | `string`: Password, or empty string if not set | Gets the password for a permission role from the config |
| `get_server_name()` | None | `string`: Server name from configuration | Gets the server name |
| `get_server_time()` | None | `number`: Server uptime in seconds | Gets the server uptime |
| `save_map(filename)` | `filename` (string): Filename to save to (optional, defaults to current map name with .saved suffix) | `boolean, string`: Success status and saved file path, or error message | Saves the current map state to a .vxl file in the maps/ directory |
| `create_explosion(x, y, z)` | `x` (number): X coordinate<br>`y` (number): Y coordinate<br>`z` (number): Z coordinate | `boolean, string`: Success status, error message | Not yet implemented |

## Gamemode System

Gamemodes are Lua scripts that define game rules and handle events. Place gamemode files in `scripts/gamemodes/`.

### Gamemode Structure

```lua
-- scripts/gamemodes/example.lua

name = "Example Gamemode"
author = "Your Name"
version = "1.0"

-- Called when gamemode is loaded
function on_init()
    broadcast_chat("Example gamemode loaded!")
    set_team_score(0, 0)
    set_team_score(1, 0)
end

-- Called every server tick for each player (60 times per second)
function on_player_update(player)
    -- Check custom conditions
end

-- Called when a player spawns
function on_player_spawn(player)
    broadcast_chat(player.name .. " spawned!")
end

-- Called when a player kills another
function on_player_kill(killer, victim)
    local team = killer.team
    local score = get_team_score(team)
    set_team_score(team, score + 1)
end

-- Called when a player joins
function on_player_join(player)
    send_chat(player.id, "Welcome to " .. get_server_name() .. "!")
end

-- Called when a player connects (before join)
function on_connect(player_id)
    -- Early connection logic
end

-- Called when a player disconnects
function on_disconnect(player_id)
    -- Cleanup
end

-- Called when a player takes damage
function on_player_damage(player, damage, source_x, source_y, source_z)
    -- Return false to cancel damage
end

-- Called when a chat message is sent
function on_chat_message(player, message)
    -- Return false to cancel message
    if message:match("badword") then
        send_chat(player.id, "Watch your language!")
        return false
    end
end

-- Called when a block is placed
function on_block_place(player, x, y, z)
    -- Return false to cancel placement
end

-- Called when a block is destroyed
function on_block_destroy(player, x, y, z)
    -- Return false to cancel destruction
end

-- Called when intel is picked up
function on_intel_pickup(player_id, team)
    -- Return false to cancel pickup
end

-- Called when intel is captured
function on_intel_capture(player_id, team)
    local score = get_team_score(team)
    set_team_score(team, score + 1)
    -- Return false to cancel capture
end

-- Called when intel is dropped
function on_intel_drop(player, team)
    -- Handle intel drop
end

-- Called when a weapon is fired
function on_weapon_fire(player)
    -- Handle weapon fire
end

-- Called when a grenade is thrown
function on_grenade_toss(player)
    -- Handle grenade toss
end

-- Called when a player restocks at tent
function on_restock(player)
    -- Handle restock
end

-- Called to check win condition
function check_win_condition()
    local team0_score = get_team_score(0)
    local team1_score = get_team_score(1)

    if team0_score >= 10 then
        return true, 0  -- Team 0 wins
    elseif team1_score >= 10 then
        return true, 1  -- Team 1 wins
    end

    return false, -1  -- No winner yet
end

-- Called to determine if map should rotate
function should_rotate_map()
    return false
end
```

### Event Hook Reference

All event hooks are optional. If not defined, default behavior is used.

| Hook | Parameters | Return Value | Description |
|------|-----------|--------------|-------------|
| `on_init()` | None | None | Called when gamemode loads |
| `on_player_update(player)` | player table | None | Called every tick per player |
| `on_player_spawn(player)` | player table | None | Player spawned |
| `on_player_kill(killer, victim)` | killer table, victim table | None | Player killed another |
| `on_player_join(player)` | player table | None | Player joined |
| `on_connect(player_id)` | player ID | None | Player connecting |
| `on_disconnect(player_id)` | player ID | None | Player disconnected |
| `on_player_damage(player, damage, sx, sy, sz)` | player table, damage number, source coords | boolean | Return false to cancel |
| `on_chat_message(player, message)` | player table, message string | boolean | Return false to cancel |
| `on_block_place(player, x, y, z)` | player table, coordinates | boolean | Return false to cancel |
| `on_block_destroy(player, x, y, z)` | player table, coordinates | boolean | Return false to cancel |
| `on_intel_pickup(player_id, team)` | player ID, team number | boolean | Return false to cancel |
| `on_intel_capture(player_id, team)` | player ID, team number | boolean | Return false to cancel |
| `on_intel_drop(player, team)` | player table, team number | None | Intel dropped |
| `on_weapon_fire(player)` | player table | None | Weapon fired |
| `on_grenade_toss(player)` | player table | None | Grenade thrown |
| `on_restock(player)` | player table | None | Player restocked |
| `check_win_condition()` | None | boolean, number | (won, winning_team) |
| `should_rotate_map()` | None | boolean | Should map rotate now |

## Command System

Commands are Lua scripts that players can execute via chat. Place command files in `scripts/commands/`.

### Command Structure

```lua
-- scripts/commands/example.lua

name = "example"
aliases = "ex,test"  -- Comma-separated aliases
description = "An example command"
usage = "/example <arg1> [arg2]"
permission = "none"  -- none, trusted, guard, moderator, admin, manager
handler = "execute"  -- Optional, defaults to "execute"

function execute(player, args)
    -- args[0] contains the command name used (helpful for aliases)
    -- args[1], args[2], etc. contain the arguments

    if #args < 1 then
        return "Usage: " .. usage
    end

    local arg1 = args[1]

    -- Do something
    broadcast_chat(player.name .. " used example command with: " .. arg1)

    -- Return a message to send back to the player
    return "Command executed successfully!"
end
```

### Command Handler Parameters

- `player` (table): The player table of the command executor
- `args` (table): Array of command arguments
  - `args[0]`: The command name or alias used (e.g., "example" or "ex")
  - `args[1]`, `args[2]`, etc.: Command arguments split by spaces

### Permission Levels

Commands can require specific permission levels:

| Permission | Description |
|-----------|-------------|
| `none` | All players can use |
| `trusted` | Trusted players and above |
| `guard` | Guards and above |
| `moderator` | Moderators and above |
| `admin` | Admins and above |
| `manager` | Managers only |

Players gain permissions by using the `/login` command with the appropriate password set in the server config.

## Constants and Enums

### Weapon Types

| Constant | Value | Description |
|----------|-------|-------------|
| `WEAPON_RIFLE` | 0 | Rifle |
| `WEAPON_SMG` | 1 | SMG |
| `WEAPON_SHOTGUN` | 2 | Shotgun |

### Tool/Item Types

| Constant | Value | Description |
|----------|-------|-------------|
| `ITEM_SPADE` | 0 | Spade |
| `ITEM_BLOCK` | 1 | Block |
| `ITEM_GUN` | 2 | Gun |
| `ITEM_GRENADE` | 3 | Grenade |

### Teams

| Constant | Value | Description |
|----------|-------|-------------|
| `TEAM_0` | 0 | Blue team |
| `TEAM_1` | 1 | Green team |

### Default Values

| Constant | Value | Description |
|----------|-------|-------------|
| `MAX_HP` | 100 | Maximum health |
| `MAX_BLOCKS` | 50 | Maximum blocks |
| `MAX_GRENADES` | 3 | Maximum grenades |
| `INITIAL_HP` | 100 | Initial health |
| `INITIAL_BLOCKS` | 50 | Initial blocks |
| `INITIAL_GRENADES` | 3 | Initial grenades |
