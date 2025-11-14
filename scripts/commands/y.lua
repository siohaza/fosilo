name = "y"
aliases = "yes"
description = "Vote yes on the current vote"
usage = "/y or /yes"
permission = "none"

function execute(player, args)
    if not has_active_vote() then
        return "There is no active vote"
    end

    local success, error_msg = cast_vote(player.id, true)

    if not success then
        return "Failed to vote: " .. (error_msg or "unknown error")
    end

    return ""
end
