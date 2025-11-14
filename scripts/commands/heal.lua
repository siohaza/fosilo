name = "heal"
aliases = ""
description = "Heal and refill a player"
usage = "/heal [player_id_or_name]"
permission = "admin"

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

    if not is_player_alive(target.id) then
        return target.name .. " is not alive"
    end

    heal_player(target.id)

    return "Healed " .. target.name
end
