name = "commands"
aliases = ""
description = "List all available commands"
usage = "/commands"
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
    return command_list_formatter(player)
end
