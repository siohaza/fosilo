name = "unban"
aliases = ""
description = "Unban a player by IP address"
usage = "/unban <ip_address>"
permission = "admin"

function execute(player, args)
    if #args < 1 then
        return "Usage: /unban <ip_address>"
    end

    local ip = args[1]

    local was_banned = is_banned(ip)
    if not was_banned then
        return "IP " .. ip .. " is not banned"
    end

    local success, error_msg = unban_ip(ip)

    if not success then
        return "Failed to unban IP: " .. (error_msg or "unknown error")
    end

    broadcast_chat("IP " .. ip .. " has been unbanned by " .. player.name)

    return "Successfully unbanned IP " .. ip
end
