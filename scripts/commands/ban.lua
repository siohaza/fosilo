name = "ban"
aliases = ""
description = "Ban a player from the server"
usage = "/ban <player> [duration] [reason]"
permission = "moderator"

function execute(player, args)
    if #args < 1 then
        return "Usage: /ban <player_id> [duration] [reason]\nDuration examples: 1h, 24h, 7d, 30d, perm (default: 24h)"
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

    local duration_hours = 24
    local reason = "Banned by admin"
    local duration_str = "24 hours"

    if #args >= 2 then
        local dur_arg = args[2]

        if dur_arg == "perm" or dur_arg == "permanent" then
            duration_hours = 0
            duration_str = "permanently"
            if #args >= 3 then
                reason = table.concat(args, " ", 3)
            end
        elseif dur_arg:match("^%d+h$") then
            duration_hours = tonumber(dur_arg:match("^(%d+)h$"))
            duration_str = duration_hours .. " hours"
            if #args >= 3 then
                reason = table.concat(args, " ", 3)
            end
        elseif dur_arg:match("^%d+d$") then
            local days = tonumber(dur_arg:match("^(%d+)d$"))
            duration_hours = days * 24
            duration_str = days .. " days"
            if #args >= 3 then
                reason = table.concat(args, " ", 3)
            end
        elseif dur_arg:match("^%d+m$") then
            local minutes = tonumber(dur_arg:match("^(%d+)m$"))
            duration_hours = minutes / 60
            duration_str = minutes .. " minutes"
            if #args >= 3 then
                reason = table.concat(args, " ", 3)
            end
        else
            reason = table.concat(args, " ", 2)
        end
    end

    local target_ip = get_player_ip(target.id)
    if target_ip == "" then
        return "Could not get player IP"
    end

    local success, error_msg = ban_player(target_ip, target.name, reason, player.name, duration_hours)

    if not success then
        return "Failed to ban player: " .. (error_msg or "unknown error")
    end

    broadcast_chat(target.name .. " was banned " .. duration_str .. ": " .. reason)

    disconnect_player(target.id, 1)

    return "Banned " .. target.name .. " (" .. target_ip .. ") " .. duration_str .. ": " .. reason
end
