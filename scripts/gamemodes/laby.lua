name = "laby"

capture_limit = 1
hog_timeout = 180
regen_rate = 1.0

flag_held = false
flag_carrier = 0
hog_timer_id = nil

tower_pos_x = 0
tower_pos_y = 0
tower_cells_x = 0
tower_cells_y = 0
tower_cells_z = 0
cell_size_x = 4
cell_size_y = 4
cell_size_z = 6

HIDE_X = 0
HIDE_Y = 0
HIDE_Z = 63

function on_init()
	set_team_score(0, 0)
	set_team_score(1, 0)
	flag_held = false
	flag_carrier = 0
	hog_timer_id = nil

	local cfg_cap = get_config_value("laby_cap_limit")
	if cfg_cap and cfg_cap > 0 then
		capture_limit = cfg_cap
	end

	local cfg_hog = get_config_value("laby_hog_timeout")
	if cfg_hog and cfg_hog > 0 then
		hog_timeout = cfg_hog
	end

	local cfg_regen = get_config_value("laby_regen_rate")
	if cfg_regen and cfg_regen > 0 then
		regen_rate = cfg_regen
	end

	tower_pos_x = get_config_value("tower_pos_x") or 0
	tower_pos_y = get_config_value("tower_pos_y") or 0
	tower_cells_x = get_config_value("tower_cells_x") or 0
	tower_cells_y = get_config_value("tower_cells_y") or 0
	tower_cells_z = get_config_value("tower_cells_z") or 0

	spawn_intel_at_random_location()

	schedule_callback(regen_rate, "regen_tick", true)
end

function spawn_intel_at_random_location()
	if tower_cells_x == 0 or tower_cells_y == 0 or tower_cells_z == 0 then
		return
	end

	local area = 20
	local center_x = tower_pos_x + math.floor(tower_cells_x * cell_size_x / 2)
	local center_y = tower_pos_y + math.floor(tower_cells_y * cell_size_y / 2)

	local max_attempts = 1000
	for attempt = 1, max_attempts do
		local x = center_x + math.random(-area, area)
		local y = center_y + math.random(-area, area)
		local lvl = math.random(0, tower_cells_z)
		local z = 63 - (3 + cell_size_z * lvl)

		local valid = true
		if attempt < max_attempts then
			if is_solid(x, y, z) or is_solid(x+1, y, z) or
			   is_solid(x, y+1, z) or is_solid(x+1, y+1, z) or
			   is_solid(x, y, z-1) or is_solid(x+1, y, z-1) or
			   is_solid(x, y+1, z-1) or is_solid(x+1, y+1, z-1) then
				valid = false
			end
		end

		if valid then
			while z < 63 and not is_solid(x, y, z) do
				z = z + 1
			end

			set_intel_position(x + 0.5, y + 0.5, z, 0)
			set_intel_position(x + 0.5, y + 0.5, z, 1)

			local floor_name = level_to_floor(lvl)
			broadcast_chat("The intel spawned " .. floor_name .. ".")
			return
		end
	end
end

function level_to_floor(lvl)
	if lvl >= tower_cells_z then
		return "on the roof"
	elseif lvl <= 0 then
		return "underground"
	elseif lvl == 1 then
		return "at ground floor"
	else
		return "at floor " .. tostring(lvl - 1)
	end
end

function z_to_floor(z)
	local lvl = math.floor((63 - z) / cell_size_z)
	return level_to_floor(lvl)
end

function regen_tick()
	for_each_player(function(player)
		if player.alive and player.hp > 0 and player.hp < 100 then
			local new_hp = math.min(player.hp + 3, 100)
			set_player_hp(player.id, new_hp)
		end
	end)
end

function on_player_spawn(player)
	if tower_cells_x == 0 or tower_cells_y == 0 or tower_cells_z == 0 then
		return
	end

	local min_x = tower_pos_x - 5
	local max_x = tower_pos_x + cell_size_x * tower_cells_x + 4
	local min_y = tower_pos_y - 5
	local max_y = tower_pos_y + cell_size_y * tower_cells_y + 4
	local num_floors = tower_cells_z

	local max_attempts = 1000
	for attempt = 1, max_attempts do
		local x = math.random(min_x, max_x)
		local y = math.random(min_y, max_y)
		local lvl = math.random(0, num_floors - 1)
		local z = 63 - (3 + cell_size_z * lvl)

		local valid = true
		if attempt < max_attempts then
			if is_solid(x, y, z) or is_solid(x+1, y, z) or
			   is_solid(x, y+1, z) or is_solid(x+1, y+1, z) or
			   is_solid(x, y, z-1) or is_solid(x+1, y, z-1) or
			   is_solid(x, y+1, z-1) or is_solid(x+1, y+1, z-1) or
			   is_solid(x, y, z-2) or is_solid(x+1, y, z-2) or
			   is_solid(x, y+1, z-2) or is_solid(x+1, y+1, z-2) then
				valid = false
			end
		end

		if valid then
			while z < 63 and not is_solid(x, y, z) do
				z = z + 1
			end

			set_player_position(player.id, x + 0.5, y + 0.5, z - 2.4)
			return
		end
	end
end

function on_player_kill(killer, victim, kill_type)
	if flag_held and victim and victim.id == flag_carrier then
		cancel_hog_timer()
		flag_held = false
		flag_carrier = 0
	end
end

function on_player_update(player)
end

function on_intel_pickup(player_id, team)
	if flag_held then
		return false
	end

	flag_held = true
	flag_carrier = player_id

	local other_team = 1 - team
	set_intel_position(HIDE_X, HIDE_Y, HIDE_Z, other_team)

	cancel_hog_timer()
	hog_timer_id = schedule_callback(hog_timeout, "check_intel_hog", false)

	return true
end

function on_intel_capture(player_id, team)
	if not flag_held or flag_carrier ~= player_id then
		return false
	end

	local player = get_player(player_id)
	if not player then
		return false
	end

	local score = get_team_score(player.team)
	set_team_score(player.team, score + 1)

	cancel_hog_timer()
	flag_held = false
	flag_carrier = 0

	spawn_intel_at_random_location()

	return true
end

function on_intel_drop(player_id, team)
	if not flag_held or flag_carrier ~= player_id then
		return
	end

	cancel_hog_timer()
	flag_held = false
	flag_carrier = 0

	local px, py, pz = get_player_position(player_id)
	if not px then
		return
	end

	local x = math.floor(px)
	local y = math.floor(py)
	local z = math.max(0, math.floor(pz))

	while z < 63 and not is_solid(x, y, z) do
		z = z + 1
	end

	set_intel_position(x + 0.5, y + 0.5, z, 0)
	set_intel_position(x + 0.5, y + 0.5, z, 1)
end

function on_grenade_explode(player, x, y, z)
	if not player or not player.alive then
		return
	end
	set_player_position(player.id, x + 0.5, y + 0.5, z - 2.4)
end

function check_intel_hog()
	hog_timer_id = nil
	if not flag_held then
		return
	end

	local player = get_player(flag_carrier)
	if player and player.alive then
		kill_player(flag_carrier)
		broadcast_chat(player.name .. " was punished for holding the intel too long")
	end

	flag_held = false
	flag_carrier = 0
end

function cancel_hog_timer()
	if hog_timer_id then
		cancel_callback(hog_timer_id)
		hog_timer_id = nil
	end
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
