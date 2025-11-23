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

        local names = {}
        for i = 1, #commands do
            table.insert(names, commands[i].name)
        end

        table.sort(names, function(a, b)
            return a:lower() < b:lower()
        end)

        local lines = {}
        local current_line = ""

        for i = 1, #names do
            local cmd = "/" .. names[i]
            local test_line = current_line
            if test_line ~= "" then
                test_line = test_line .. ", " .. cmd
            else
                test_line = cmd
            end

            if #test_line > 60 then
                if current_line ~= "" then
                    table.insert(lines, current_line)
                end
                current_line = cmd
            else
                current_line = test_line
            end
        end

        if current_line ~= "" then
            table.insert(lines, current_line)
        end

        for i = 1, #lines do
            send_chat(player.id, lines[i])
        end

        return "Type /help <command> for details"
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
