name = "arena"

score_limit = 5
max_round_time = 180
spawn_zone_time = 15
timeout_is_draw = false
sudden_death_enabled = false
is_sudden_death = false

ROUND_STATE_COUNTDOWN = 0
ROUND_STATE_ACTIVE = 1
ROUND_STATE_ENDED = 2

round_state = ROUND_STATE_COUNTDOWN
team1_alive = 0
team2_alive = 0
round_start_time = 0
countdown_start_time = 0
countdown_timer = nil
round_timer = nil

function on_init()
	set_team_score(0, 0)
	set_team_score(1, 0)
	round_state = ROUND_STATE_COUNTDOWN
	team1_alive = 0
	team2_alive = 0
	is_sudden_death = false

	local config_score_limit = get_config_value("arena_score_limit")
	if config_score_limit and config_score_limit > 0 then
		score_limit = config_score_limit
	end

	local config_timeout_draw = get_config_value("timeout_is_draw")
	if config_timeout_draw ~= nil then
		timeout_is_draw = config_timeout_draw
	end

	local config_sudden_death = get_config_value("sudden_death_enabled")
	if config_sudden_death ~= nil then
		sudden_death_enabled = config_sudden_death
	end

	start_countdown()
end

function start_countdown()
	round_state = ROUND_STATE_COUNTDOWN
	broadcast_chat("Round starting in " .. spawn_zone_time .. " seconds!")
	countdown_timer = schedule_callback(spawn_zone_time, "start_round", false)
end

function start_round()
	round_state = ROUND_STATE_ACTIVE
	broadcast_chat("Round started! Fight!")
	round_timer = schedule_callback(max_round_time, "end_round_timeout", false)
	schedule_callback(1, "check_round_end", true)
end

function end_round_timeout()
	if round_state == ROUND_STATE_ACTIVE then
		if timeout_is_draw then
			broadcast_chat("Round ended in a draw due to timeout!")
			round_state = ROUND_STATE_ENDED
			schedule_callback(5, "start_countdown", false)
		else
			broadcast_chat("Round ended due to timeout!")
			determine_round_winner()
		end
	end
end

function check_round_end()
	if round_state ~= ROUND_STATE_ACTIVE then
		return
	end

	team1_alive = 0
	team2_alive = 0

	for i = 0, get_player_count() - 1 do
		local p = get_player(i)
		if p and p.alive then
			if p.team == 0 then
				team1_alive = team1_alive + 1
			elseif p.team == 1 then
				team2_alive = team2_alive + 1
			end
		end
	end

	if team1_alive == 0 and team2_alive == 0 then
		broadcast_chat("Round ended in a draw!")
		round_state = ROUND_STATE_ENDED
		if round_timer then
			cancel_callback(round_timer)
		end
		schedule_callback(5, "start_countdown", false)
	elseif team1_alive == 0 then
		broadcast_chat("Green team wins the round!")
		round_state = ROUND_STATE_ENDED
		local score = get_team_score(1)
		set_team_score(1, score + 1)
		if round_timer then
			cancel_callback(round_timer)
		end
		check_sudden_death()
		schedule_callback(5, "start_countdown", false)
	elseif team2_alive == 0 then
		broadcast_chat("Blue team wins the round!")
		round_state = ROUND_STATE_ENDED
		local score = get_team_score(0)
		set_team_score(0, score + 1)
		if round_timer then
			cancel_callback(round_timer)
		end
		check_sudden_death()
		schedule_callback(5, "start_countdown", false)
	end
end

function check_sudden_death()
	if not sudden_death_enabled then
		return
	end

	local team1_score = get_team_score(0)
	local team2_score = get_team_score(1)

	if team1_score == score_limit - 1 and team2_score == score_limit - 1 then
		if not is_sudden_death then
			is_sudden_death = true
			broadcast_chat("SUDDEN DEATH! Next round wins the match!")
		end
	end
end

function determine_round_winner()
	if team1_alive > team2_alive then
		broadcast_chat("Blue team wins by having more players alive!")
		local score = get_team_score(0)
		set_team_score(0, score + 1)
		check_sudden_death()
	elseif team2_alive > team1_alive then
		broadcast_chat("Green team wins by having more players alive!")
		local score = get_team_score(1)
		set_team_score(1, score + 1)
		check_sudden_death()
	else
		broadcast_chat("Round ended in a draw!")
	end

	round_state = ROUND_STATE_ENDED
	schedule_callback(5, "start_countdown", false)
end

function on_player_spawn(player)
	if round_state == ROUND_STATE_ACTIVE then
		kill_player(player.id)
		broadcast_chat(player.name .. " will respawn next round")
		return false
	end

	if player.team == 0 then
		team1_alive = team1_alive + 1
	elseif player.team == 1 then
		team2_alive = team2_alive + 1
	end

	return true
end

function on_player_kill(killer, victim, kill_type)
	if round_state == ROUND_STATE_ACTIVE and victim then
		if victim.team == 0 then
			team1_alive = team1_alive - 1
		else
			team2_alive = team2_alive - 1
		end
	end
end

function on_player_update(player)
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

	if team1_score >= score_limit then
		return true, 0
	end

	if team2_score >= score_limit then
		return true, 1
	end

	return false, 0
end

function should_rotate_map()
	local won, _ = check_win_condition()
	return won
end
