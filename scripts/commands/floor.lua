name = "floor"
aliases = "f"
description = "Show your current floor and the intel's floor"
usage = "/floor"
permission = "none"

function execute(player, args)
	local num_floors = get_config_value("tower_cells_z") or 0
	if num_floors == 0 then
		return "Floor info not available (no tower map loaded)"
	end

	local cell_z = 6
	local px, py, pz = get_player_position(player.id)
	if not px then
		return ""
	end

	local player_floor = z_to_floor(pz, cell_z, num_floors)

	local ix0, iy0, iz0 = get_intel_position(0)
	local ix1, iy1, iz1 = get_intel_position(1)

	local ix, iy, iz = ix0, iy0, iz0
	if ix0 and ix0 < 1 then
		ix, iy, iz = ix1, iy1, iz1
	end

	if not ix then
		return "You are " .. player_floor .. "."
	end

	local intel_floor = z_to_floor(iz, cell_z, num_floors)
	return "You are " .. player_floor .. ", the intel is " .. intel_floor .. "."
end

function z_to_floor(z, cell_z, num_floors)
	local lvl = math.floor((63 - z) / cell_z)
	if lvl >= num_floors then
		return "on the roof"
	elseif lvl <= 0 then
		return "underground"
	elseif lvl == 1 then
		return "at ground floor"
	else
		return "at floor " .. tostring(lvl - 1)
	end
end
