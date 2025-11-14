local SPECTATOR_TEAM = 255

name = "switch"
aliases = "team"
description = "Switch a player's team"
usage = "/switch <player_id_or_name> [team]"
permission = "admin"

function execute(player, args)
    if #args < 1 then
        return "Usage: /switch <player_id_or_name> [team]"
    end

    local target_arg = args[1]

    if target_arg:sub(1,1) == "#" then
        target_arg = target_arg:sub(2)
    end

    local target_id = tonumber(target_arg)
    local target

    if target_id then
        target = get_player(target_id)
    else
        target = get_player_by_name(target_arg)
    end

    if not target then
        return "Player not found: " .. args[1]
    end

    local new_team = -1
    if #args >= 2 then
        local team_arg = args[2]:lower()
        if team_arg == "blue" or team_arg == "0" then
            new_team = 0
        elseif team_arg == "green" or team_arg == "1" then
            new_team = 1
        elseif team_arg == "spectator" or team_arg == "spec" then
            new_team = SPECTATOR_TEAM
        end
    else
        if target.team == 0 then
            new_team = 1
        else
            new_team = 0
        end
    end

    if new_team == -1 then
        return "Invalid team. Use: blue/0, green/1, or spectator/spec"
    end

    set_player_team(target.id, new_team)

    local team_names = {
        [0] = "blue",
        [1] = "green",
        [SPECTATOR_TEAM] = "spectator"
    }

    local team_name = team_names[new_team] or tostring(new_team)
    return "Switched " .. target.name .. " to " .. team_name .. " team"
end
