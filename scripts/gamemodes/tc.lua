name = "tc"

max_score = 50
capture_distance = 16.0
capture_rate = 0.05
score_interval = 5.0

territories = {}
territory_count = 0
last_score_time = 0

function on_init()
	set_team_score(0, 0)
	set_team_score(1, 0)
	last_score_time = 0

	local config_max_score = get_config_value("tc_max_score")
	if config_max_score and config_max_score > 0 then
		max_score = config_max_score
	end

	local config_capture_distance = get_config_value("tc_capture_distance")
	if config_capture_distance and config_capture_distance > 0 then
		capture_distance = config_capture_distance
	end

	local config_capture_rate = get_config_value("tc_capture_rate")
	if config_capture_rate and config_capture_rate > 0 then
		capture_rate = config_capture_rate
	end

	init_territories()
end

function init_territories()
	local map_width = get_map_width()
	local map_height = get_map_height()

	local center_x = map_width / 2
	local center_y = map_height / 2

	local z = find_top_block(center_x, center_y)

	territories[1] = {
		x = center_x,
		y = center_y,
		z = z,
		team = nil,
		progress = 0.5,
		players_team0 = 0,
		players_team1 = 0,
		id = 0
	}

	territory_count = 1
end

function on_player_spawn(player)
end

function on_player_kill(killer, victim, kill_type)
end

function on_player_update(player)
	local current_time = get_server_time()

	if current_time - last_score_time >= score_interval then
		award_territory_points()
		last_score_time = current_time
	end

	if not player.alive then
		return
	end

	local pos = player.position
	if not pos then
		return
	end

	local px, py, pz = pos[1], pos[2], pos[3]

	for i = 1, territory_count do
		local territory = territories[i]
		local dx = px - territory.x
		local dy = py - territory.y
		local dz = pz - territory.z
		local distance = math.sqrt(dx*dx + dy*dy + dz*dz)

		if distance <= capture_distance then
			update_territory_capture(territory, player)
		end
	end
end

function award_territory_points()
	for i = 1, territory_count do
		local territory = territories[i]
		if territory.team ~= nil then
			local score = get_team_score(territory.team)
			set_team_score(territory.team, score + 1)
		end
	end
end

function update_territory_capture(territory, player)
	local team = player.team
	if team >= 2 then
		return
	end

	local old_progress = territory.progress

	if territory.team == nil then
		if team == 0 then
			territory.progress = territory.progress - capture_rate / 60
		else
			territory.progress = territory.progress + capture_rate / 60
		end

		if territory.progress <= 0.0 then
			territory.progress = 0.0
			capture_territory(territory, 0, player.id)
		elseif territory.progress >= 1.0 then
			territory.progress = 1.0
			capture_territory(territory, 1, player.id)
		else
			send_progress_bar(territory.id, team, 1, territory.progress)
		end
	elseif territory.team ~= team then
		if team == 0 then
			territory.progress = territory.progress - capture_rate / 60
		else
			territory.progress = territory.progress + capture_rate / 60
		end

		if territory.progress <= 0.0 and territory.team == 1 then
			territory.team = nil
			territory.progress = 0.5
			send_progress_bar(territory.id, 255, 0, 0.5)
		elseif territory.progress >= 1.0 and territory.team == 0 then
			territory.team = nil
			territory.progress = 0.5
			send_progress_bar(territory.id, 255, 0, 0.5)
		else
			send_progress_bar(territory.id, team, 1, territory.progress)
		end
	end
end

function capture_territory(territory, team, player_id)
	territory.team = team

	local score = get_team_score(team)
	set_team_score(team, score + 1)

	local team_name = team == 0 and "Blue" or "Green"
	broadcast_chat("Territory captured by " .. team_name .. " team!")

	send_territory_capture(player_id, territory.id, 1, team)
	send_progress_bar(territory.id, team, 0, territory.progress)
end

function on_intel_pickup(player_id, team)
	return false
end

function on_intel_capture(player_id, team)
	return false
end

function check_win_condition()
	local team1_score = get_team_score(0)
	local team2_score = get_team_score(1)

	if team1_score >= max_score then
		return true, 0
	end

	if team2_score >= max_score then
		return true, 1
	end

	return false, 0
end

function should_rotate_map()
	local won, _ = check_win_condition()
	return won
end
