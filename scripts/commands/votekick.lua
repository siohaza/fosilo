name = "votekick"
aliases = "vk"
description = "Start a vote to kick a player"
usage = "/votekick <player_id_or_name> <reason>"
permission = "none"

function execute(player, args)
    if #args < 2 then
        return "Usage: /votekick <player_id_or_name> <reason>"
    end

    if has_active_vote() then
        return "There is already a vote in progress"
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

    if target.id == player.id then
        return "You cannot votekick yourself"
    end

    local reason = table.concat(args, " ", 2)

    local success, error_msg = start_votekick(player.id, target.id, reason)

    if not success then
        return "Failed to start votekick: " .. error_msg
    end

    return ""
end
