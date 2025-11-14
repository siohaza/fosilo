name = "help"
aliases = "h,?"
description = "Show help for commands"
usage = "/help [command]"
permission = "none"

if command_list_formatter == nil then
    function command_list_formatter(player)
        local commands = get_available_commands(player.id)

        if #commands == 0 then
            return "No commands available"
        end

        local entries = {}
        for i = 1, #commands do
            local cmd = commands[i]
            local entry = "/" .. cmd.name

            if cmd.aliases and #cmd.aliases > 0 then
                local alias_list = {}
                for j = 1, #cmd.aliases do
                    local alias = cmd.aliases[j]
                    if alias ~= nil and alias ~= "" then
                        table.insert(alias_list, "/" .. alias)
                    end
                end
                if #alias_list > 0 then
                    entry = entry .. " (aliases: " .. table.concat(alias_list, ", ") .. ")"
                end
            end

            table.insert(entries, entry)
        end

        table.sort(entries, function(a, b)
            return a:lower() < b:lower()
        end)

        return "Available commands: " .. table.concat(entries, ", ") .. " - Use /help <command> for more info"
    end
end

function execute(player, args)
    local commands = get_available_commands(player.id)

    if #args == 0 then
        return command_list_formatter(player)
    end

    local cmd_name = args[1]:lower()

    for i = 1, #commands do
        local cmd = commands[i]
        if cmd.name == cmd_name then
            local help_text = cmd.usage
            if cmd.description and cmd.description ~= "" then
                help_text = help_text .. " - " .. cmd.description
            end

            if cmd.aliases and #cmd.aliases > 0 then
                local alias_list = {}
                for j = 1, #cmd.aliases do
                    if cmd.aliases[j] ~= "" then
                        table.insert(alias_list, cmd.aliases[j])
                    end
                end
                if #alias_list > 0 then
                    help_text = help_text .. " (aliases: " .. table.concat(alias_list, ", ") .. ")"
                end
            end

            return help_text
        end

        if cmd.aliases then
            for j = 1, #cmd.aliases do
                if cmd.aliases[j] == cmd_name then
                    local help_text = cmd.usage
                    if cmd.description and cmd.description ~= "" then
                        help_text = help_text .. " - " .. cmd.description
                    end
                    return help_text
                end
            end
        end
    end

    return "Unknown command: " .. cmd_name .. " - Type /help for available commands"
end
