name = "kill"
aliases = ""
description = "Kill yourself or another player"
usage = "/kill [player_id_or_name]"
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

        if not has_permission(player.id, "admin") and target.id ~= player.id then
            return "You can only kill yourself"
        end
    end

    if not is_player_alive(target.id) then
        return target.name .. " is already dead"
    end

    kill_player(target.id)

    if target.id == player.id then
        return "You killed yourself"
    else
        return "Killed " .. target.name
    end
end
