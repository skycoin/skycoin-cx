-- Walks 100 points 80000 times.
-- Needs Lua 5.4 or later for 64 bit integers and bit operators.

local FNV_OFFSET = 0xcbf29ce484222325 -- 14695981039346656037
local FNV_PRIME = 1099511628211

-- fnv folds one 64 bit word. Lua integers wrap like a ui64.
local function fnv(h, v)
	return (h ~ v) * FNV_PRIME
end

-- report prints the hash as an unsigned 64 bit number and the pass line.
local function report(name, h, ok)
	if h >= 0 then
		print(string.format("%d", h))
	else
		local q = (h >> 1) // 5
		print(string.format("%d%d", q, h - q * 10))
	end
	print(name .. (ok and " pass" or " FAIL"))
	return ok and 0 or 1
end

local STRUCT_ROUNDS = 80000

local function main()
	local h = FNV_OFFSET
	local ok = true
	local points = {}
	for i = 0, 99 do
		points[i + 1] = { x = i, y = 2 * i }
	end
	for _ = 1, STRUCT_ROUNDS do
		local total = 0
		for i = 1, 100 do
			local p = points[i]
			total = total + p.x + p.y
		end
		h = fnv(h, total)
		ok = ok and total == 14850
	end
	return report("struct", h, ok)
end

os.exit(main())
