name = "client"
aliases = "cli,clin,client_info"
description = "Show client information for a player"
usage = "/client [player_id_or_name]"
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
            target = get_player_by_id(target_id)
            if not target then
                return "Player #" .. target_id .. " not found"
            end
        else
            target = get_player_by_name(target_arg)
            if not target then
                return "Player '" .. target_arg .. "' not found"
            end
        end
    end

    local client_name = get_client_name(target.client_identifier)
    local version_str = ""

    if target.version_major ~= 0 or target.version_minor ~= 0 or target.version_revision ~= 0 then
        version_str = string.format(" v%d.%d.%d", target.version_major, target.version_minor, target.version_revision)
    end

    local os_str = ""
    if target.os_info and target.os_info ~= "" then
        os_str = " on " .. target.os_info
    end

    local player_name = target.name
    if target.id ~= player.id then
        player_name = player_name .. " (#" .. target.id .. ")"
    else
        player_name = "You"
    end

    return player_name .. " connected with " .. client_name .. version_str .. os_str
end

function get_client_name(identifier)
    if identifier == "o" then
        return "OpenSpades"
    elseif identifier == "B" then
        return "BetterSpades"
    elseif identifier == "a" then
        return "ACE"
    else
        return "Voxlap"
    end
end
