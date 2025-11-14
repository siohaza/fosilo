name = "mapname"
aliases = "map"
description = "Show current map name"
usage = "/mapname"
permission = "none"

function execute(player, args)
    return "Current map: " .. get_map_name()
end
