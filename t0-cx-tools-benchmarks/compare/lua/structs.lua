-- Fills and reads a struct of two 100000 element arrays 60 times.
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

local SIZE = 100000
local bm = { a = {}, b = {} }

local function main()
	local h = FNV_OFFSET
	local ok = true
	for round = 0, 59 do
		for c = 0, SIZE - 1 do
			bm.a[c + 1] = c + round
			bm.b[c + 1] = c * 3
		end
		local sum = 0
		for c = 1, SIZE do
			sum = sum + bm.a[c] + bm.b[c]
		end
		h = fnv(h, sum)
		ok = ok and sum == 19999800000 + 100000 * round
	end
	return report("structs", h, ok)
end

os.exit(main())
