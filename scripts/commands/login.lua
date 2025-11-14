name = "login"
aliases = "auth"
description = "Login with a role password to gain permissions"
usage = "/login <role> <password>"
permission = "none"

function execute(player, args)
    if #args < 2 then
        return "Usage: /login <role> <password>"
    end

    local role = string.lower(args[1])
    local password = args[2]

    local valid_roles = {"trusted", "guard", "moderator", "mod", "admin", "manager"}
    local role_valid = false
    for _, v in ipairs(valid_roles) do
        if v == role then
            role_valid = true
            break
        end
    end

    if not role_valid then
        return "Invalid role. Valid roles: trusted, guard, moderator, admin, manager"
    end

    local config_password = get_config_password(role)

    if config_password == "" then
        return "No password set for role: " .. role
    end

    if password ~= config_password then
        return "Incorrect password"
    end

    local success, err = set_player_permission(player.id, role)
    if not success then
        return "Failed to set permissions: " .. err
    end

    local display_role = role
    if role == "mod" then
        display_role = "moderator"
    end

    return "Successfully logged in as " .. display_role
end
