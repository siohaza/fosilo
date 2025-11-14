name = "babel"

capture_limit = 10
reverse_mode = false
regenerate_tower = false
regeneration_rate = 1.0
regeneration_timer = 0

flag_held = false
flag_carrier = 0

tower_blocks = {}

function on_init()
	set_team_score(0, 0)
	set_team_score(1, 0)
	flag_held = false
	flag_carrier = 0
	tower_blocks = {}
	regeneration_timer = 0

	local config_capture_limit = get_config_value("babel_capture_limit")
	if config_capture_limit and config_capture_limit > 0 then
		capture_limit = config_capture_limit
	end

	local config_reverse = get_config_value("babel_reverse")
	if config_reverse ~= nil then
		reverse_mode = config_reverse
	end

	local config_regenerate = get_config_value("regenerate_tower")
	if config_regenerate ~= nil then
		regenerate_tower = config_regenerate
	end

	local config_regen_rate = get_config_value("regeneration_rate")
	if config_regen_rate and config_regen_rate > 0 then
		regeneration_rate = config_regen_rate
	end

	if regenerate_tower then
		store_tower_blocks()
	end
end

function store_tower_blocks()
	local map_width = get_map_width()
	local map_height = get_map_height()

	local center_x = math.floor(map_width / 2)
	local center_y = math.floor(map_height / 2)
	local tower_radius = 20

	for x = center_x - tower_radius, center_x + tower_radius do
		for y = center_y - tower_radius, center_y + tower_radius do
			local dx = x - center_x
			local dy = y - center_y
			if dx * dx + dy * dy <= tower_radius * tower_radius then
				for z = 0, 62 do
					local block = get_block(x, y, z)
					if block and block > 0 then
						table.insert(tower_blocks, {x = x, y = y, z = z, color = block})
					end
				end
			end
		end
	end
end

function on_player_spawn(player)
end

function on_player_kill(killer, victim, kill_type)
end

function on_player_update(player)
	if regenerate_tower and #tower_blocks > 0 then
		local current_time = get_game_time()
		if current_time - regeneration_timer >= regeneration_rate then
			regenerate_destroyed_blocks()
			regeneration_timer = current_time
		end
	end
end

function regenerate_destroyed_blocks()
	local blocks_regenerated = 0
	local max_per_cycle = 10

	for i = 1, #tower_blocks do
		if blocks_regenerated >= max_per_cycle then
			break
		end

		local block = tower_blocks[i]
		local current_block = get_block(block.x, block.y, block.z)

		if not current_block or current_block == 0 then
			set_block(block.x, block.y, block.z, block.color)
			blocks_regenerated = blocks_regenerated + 1
		end
	end
end

function on_intel_pickup(player_id, team)
	flag_held = true
	flag_carrier = player_id
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

	local player_team = player.team

	if reverse_mode then
		if player_team == team then
			local score = get_team_score(player_team)
			set_team_score(player_team, score + 1)
			flag_held = false
			flag_carrier = 0
			return true
		end
	else
		if player_team ~= team then
			local score = get_team_score(player_team)
			set_team_score(player_team, score + 1)
			flag_held = false
			flag_carrier = 0
			return true
		end
	end

	return false
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
