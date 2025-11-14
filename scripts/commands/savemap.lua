name = "savemap"
aliases = ""
description = "Save the current map state to a .vxl file"
usage = "/savemap [filename]"
permission = "admin"

function execute(player, args)
    local filename = ""

    if #args > 0 then
        filename = table.concat(args, " ")
    end

    local success, result = save_map(filename)

    if success then
        return "Map saved to: " .. result
    else
        return "Failed to save map: " .. result
    end
end
