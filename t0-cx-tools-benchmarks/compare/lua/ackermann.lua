-- Ackermann(3, 5) 350 times, 42,438 calls each.
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

local function ackermann(m, n)
	if m == 0 then
		return n + 1
	end
	if n == 0 then
		return ackermann(m - 1, 1)
	end
	return ackermann(m - 1, ackermann(m, n - 1))
end

local function main()
	local h = FNV_OFFSET
	local ok = true
	for _ = 1, 350 do
		local r = ackermann(3, 5)
		h = fnv(h, r)
		ok = ok and r == 253
	end
	return report("ackermann", h, ok)
end

os.exit(main())
