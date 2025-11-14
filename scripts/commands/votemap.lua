name = "votemap"
aliases = "vm"
description = "Start a vote to change the map"
usage = "/votemap"
permission = "none"

function execute(player, args)
    if has_active_vote() then
        return "There is already a vote in progress"
    end

    local success, error_msg = start_votemap(player.id)

    if not success then
        return "Failed to start votemap: " .. error_msg
    end

    return ""
end
