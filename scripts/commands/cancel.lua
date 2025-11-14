name = "cancel"
aliases = {}
description = "Cancel the current vote (instigator or admin only)"
usage = "/cancel"
permission = "none"

function execute(player, args)
    if not has_active_vote() then
        return "There is no active vote to cancel"
    end

    local success, error_msg = cancel_vote(player.id)

    if not success then
        return "Failed to cancel vote: " .. error_msg
    end

    return ""
end
