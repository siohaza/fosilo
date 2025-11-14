name = "ping"
aliases = ""
description = "Show player ping/latency"
usage = "/ping [player_id_or_name]"
permission = "none"

function execute(player, args)
    local target = player

    if #args > 0 then
        local target_arg = args[1]

        if target_arg:sub(1,1) == "#" then
            target_arg = target_arg:sub(2)
        end

        local target_id = tonumber(target_arg)

        if target_id then
            target = get_player(target_id)
        else
            target = get_player_by_name(target_arg)
        end

        if not target then
            return "Player not found: " .. args[1]
        end
    end

    local ping = get_player_ping(target.id)
    if ping < 0 then
        return target.name .. "'s ping: unavailable"
    end
    return target.name .. "'s ping: " .. ping .. "ms"
end
