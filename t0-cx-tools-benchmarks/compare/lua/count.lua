-- Sums 0 to 29,999,999.
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

local COUNT_N = 30000000

local function main()
	local total = 0
	for i = 0, COUNT_N - 1 do
		total = total + i
	end
	return report("count", fnv(FNV_OFFSET, total), total == COUNT_N * (COUNT_N - 1) // 2)
end

os.exit(main())
