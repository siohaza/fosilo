name = "kick"
aliases = "k"
description = "Kick a player from the server"
usage = "/kick <player_id_or_name> [reason]"
permission = "moderator"

function execute(player, args)
    if #args < 1 then
        return "Usage: /kick <player_id_or_name> [reason]"
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

    local reason = "Kicked by moderator"
    if #args >= 2 then
        reason = table.concat(args, " ", 2)
    end

    broadcast_chat(target.name .. " was kicked: " .. reason)
    kick_player_cmd(target.id, reason)

    return "Kicked " .. target.name .. ": " .. reason
end
