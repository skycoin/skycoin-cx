-- Bubble sorts a reversed array of 150 400 times.
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

local ARRAY_ROUNDS = 400

local function main()
	local h = FNV_OFFSET
	local ok = true
	local nums = {}
	for _ = 1, ARRAY_ROUNDS do
		for i = 1, 150 do
			nums[i] = 151 - i
		end
		local swaps = 0
		for i = 0, 148 do
			for j = 1, 149 - i do
				if nums[j] > nums[j + 1] then
					nums[j], nums[j + 1] = nums[j + 1], nums[j]
					swaps = swaps + 1
				end
			end
		end
		h = fnv(h, swaps)
		ok = ok and swaps == 11175
	end
	for i = 1, 150 do
		h = fnv(h, nums[i])
		ok = ok and nums[i] == i
	end
	return report("array", h, ok)
end

os.exit(main())
