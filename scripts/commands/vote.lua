name = "vote"
aliases = "y,n,yes,no"
description = "Cast a vote in the current poll"
usage = "/vote <yes|no|y|n|1-N> or /y or /n"
permission = "none"

function execute(player, args)
    if not has_active_vote() then
        return "There is no active vote"
    end

    local vote_type = get_vote_type()
    local choice = nil
    local cmd_name = args[0]

    if vote_type == "kick" then
        if cmd_name == "y" or cmd_name == "yes" then
            choice = true
        elseif cmd_name == "n" or cmd_name == "no" then
            choice = false
        elseif #args > 0 then
            local arg = args[1]:lower()
            if arg == "y" or arg == "yes" or arg == "1" then
                choice = true
            elseif arg == "n" or arg == "no" or arg == "0" then
                choice = false
            else
                return "Invalid vote choice for votekick. Use: yes/y/1 or no/n/0"
            end
        else
            return "Usage: /vote <yes|no|y|n> or use /y or /n"
        end
    elseif vote_type == "map" then
        if #args > 0 then
            local arg = args[1]
            local choice_num = tonumber(arg)
            if choice_num then
                choice = choice_num
            else
                local map_choices = get_vote_choices()
                if map_choices then
                    for i, map_name in pairs(map_choices) do
                        if map_name:lower() == arg:lower() then
                            choice = map_name
                            break
                        end
                    end
                    if not choice then
                        return "Invalid map choice. Use /vote <number> or /vote <mapname>"
                    end
                else
                    return "Could not get map choices"
                end
            end
        else
            return "Usage: /vote <number> or /vote <mapname>"
        end
    else
        return "Unknown vote type"
    end

    if choice == nil then
        return "Invalid vote choice"
    end

    local success, error_msg = cast_vote(player.id, choice)

    if not success then
        return "Failed to vote: " .. error_msg
    end

    return ""
end
