name = "reloadgm"
aliases = "rlgm"
description = "Reload the current gamemode without restarting the server"
usage = "/reloadgm"
permission = "admin"

function execute(player, args)
    local success, error_msg = reload_gamemode()

    if not success then
        return "Failed to reload gamemode: " .. error_msg
    end

    return "Successfully reloaded gamemode"
end
