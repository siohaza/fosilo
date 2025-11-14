name = "ctf"

capture_limit = 10
intel_reset_time = 30.0
capture_time_bonus = 0

intel_carriers = {}
intel_drop_times = {}
intel_pickup_times = {}

function on_init()
	set_team_score(0, 0)
	set_team_score(1, 0)
	intel_carriers = {[0] = nil, [1] = nil}
	intel_drop_times = {[0] = 0, [1] = 0}
	intel_pickup_times = {[0] = 0, [1] = 0}

	local config_capture_limit = get_config_value("capture_limit")
	if config_capture_limit and config_capture_limit > 0 then
		capture_limit = config_capture_limit
	end

	local config_flag_return_time = get_config_value("flag_return_time")
	if config_flag_return_time and config_flag_return_time > 0 then
		intel_reset_time = config_flag_return_time
	end

	local config_capture_time_bonus = get_config_value("capture_time_bonus")
	if config_capture_time_bonus and config_capture_time_bonus > 0 then
		capture_time_bonus = config_capture_time_bonus
	end
end

function on_player_spawn(player)
end

function on_player_kill(killer, victim, kill_type)
	if victim and victim.has_intel then
		for team = 0, 1 do
			if intel_carriers[team] == victim.id then
				intel_carriers[team] = nil
			end
		end
	end
end

function on_player_update(player)
	local current_time = get_game_time()

	for team = 0, 1 do
		if intel_drop_times[team] > 0 and intel_carriers[team] == nil then
			if current_time - intel_drop_times[team] >= intel_reset_time then
				reset_intel_to_base(team)
				intel_drop_times[team] = 0
			end
		end
	end
end

function reset_intel_to_base(team)
	local base_pos = get_base_position(team)
	if base_pos then
		set_intel_position(base_pos[1], base_pos[2], base_pos[3])
		local team_name = team == 0 and "Blue" or "Green"
		broadcast_chat(team_name .. " team intel returned to base!")
	end
end

function on_intel_pickup(player_id, team)
	intel_carriers[team] = player_id
	intel_drop_times[team] = 0
	intel_pickup_times[team] = get_game_time()
	return true
end

function on_intel_drop(player_id, team)
	intel_carriers[team] = nil
	intel_drop_times[team] = get_game_time()
	intel_pickup_times[team] = 0
	return true
end

function on_intel_capture(player_id, team)
	local score = get_team_score(team)
	local points_to_add = 1

	if capture_time_bonus > 0 and intel_pickup_times[team] > 0 then
		local current_time = get_game_time()
		local capture_time = current_time - intel_pickup_times[team]

		if capture_time > 0 and capture_time <= capture_time_bonus then
			points_to_add = 2
			local player = get_player(player_id)
			if player then
				local player_name = player.name
				local team_name = team == 0 and "Blue" or "Green"
				broadcast_chat(player_name .. " captured the " .. team_name .. " intel quickly! +2 points!")
			end
		end
	end

	set_team_score(team, score + points_to_add)

	intel_carriers[team] = nil
	intel_drop_times[team] = 0
	intel_pickup_times[team] = 0

	return true
end

function check_win_condition()
	local team1_score = get_team_score(0)
	local team2_score = get_team_score(1)

	if team1_score >= capture_limit then
		return true, 0
	end

	if team2_score >= capture_limit then
		return true, 1
	end

	return false, 0
end

function should_rotate_map()
	local won, _ = check_win_condition()
	return won
end
