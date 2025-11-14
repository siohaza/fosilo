name = "tdm"

kill_limit = 100
intel_points = 10
remove_intel = false
headshot_multiplier = 0
enable_killstreaks = false

player_killstreaks = {}

function on_init()
	set_team_score(0, 0)
	set_team_score(1, 0)
	player_killstreaks = {}

	local config_kill_limit = get_config_value("kill_limit")
	if config_kill_limit and config_kill_limit > 0 then
		kill_limit = config_kill_limit
	end

	local config_intel_points = get_config_value("intel_points")
	if config_intel_points and config_intel_points > 0 then
		intel_points = config_intel_points
	end

	local config_remove_intel = get_config_value("remove_intel")
	if config_remove_intel ~= nil then
		remove_intel = config_remove_intel
	end

	local config_headshot_mult = get_config_value("headshot_multiplier")
	if config_headshot_mult and config_headshot_mult > 0 then
		headshot_multiplier = config_headshot_mult
	end

	local config_killstreaks = get_config_value("enable_killstreaks")
	if config_killstreaks ~= nil then
		enable_killstreaks = config_killstreaks
	end
end

function on_player_spawn(player)
	if player and player.id then
		if not player_killstreaks[player.id] then
			player_killstreaks[player.id] = 0
		end
	end
end

function on_player_kill(killer, victim, kill_type)
	if killer and victim and killer.team ~= victim.team then
		local score = get_team_score(killer.team)
		local points = 1

		if kill_type == 1 and headshot_multiplier > 0 then
			points = headshot_multiplier
			broadcast_chat(killer.name .. " got a headshot on " .. victim.name .. "!")
		end

		set_team_score(killer.team, score + points)

		if enable_killstreaks and killer.id then
			if not player_killstreaks[killer.id] then
				player_killstreaks[killer.id] = 0
			end

			player_killstreaks[killer.id] = player_killstreaks[killer.id] + 1
			local streak = player_killstreaks[killer.id]

			if streak == 5 then
				broadcast_chat(killer.name .. " is on a 5 kill streak!")
			elseif streak == 10 then
				broadcast_chat(killer.name .. " is on a 10 kill streak!")
				local new_score = get_team_score(killer.team)
				set_team_score(killer.team, new_score + 2)
			elseif streak == 15 then
				broadcast_chat(killer.name .. " is on a 15 kill streak!")
				local new_score = get_team_score(killer.team)
				set_team_score(killer.team, new_score + 3)
			elseif streak >= 20 and streak % 5 == 0 then
				broadcast_chat(killer.name .. " is on a " .. streak .. " kill streak!")
				local new_score = get_team_score(killer.team)
				set_team_score(killer.team, new_score + 5)
			end
		end
	end

	if victim and victim.id and player_killstreaks[victim.id] then
		player_killstreaks[victim.id] = 0
	end
end

function on_player_update(player)
end

function on_intel_pickup(player_id, team)
	return not remove_intel
end

function on_intel_capture(player_id, team)
	if remove_intel then
		return false
	end

	local score = get_team_score(team)
	set_team_score(team, score + intel_points)

	return true
end

function check_win_condition()
	local team1_score = get_team_score(0)
	local team2_score = get_team_score(1)

	if team1_score >= kill_limit then
		return true, 0
	end

	if team2_score >= kill_limit then
		return true, 1
	end

	return false, 0
end

function should_rotate_map()
	local won, _ = check_win_condition()
	return won
end
