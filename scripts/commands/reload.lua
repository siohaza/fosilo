name = "reload"
aliases = "rlcmd"
description = "Reload all Lua commands without restarting the server"
usage = "/reload"
permission = "admin"

function execute(player, args)
    local success, error_msg = reload_commands()

    if not success then
        return "Failed to reload commands: " .. error_msg
    end

    return "Successfully reloaded all Lua commands"
end
