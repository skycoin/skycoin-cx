-- n! for n 0 to 20 with a loop, 120000 times.
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

local function factorial(n)
	local result = 1
	for k = 2, n do
		result = result * k
	end
	return result
end

local function main()
	local h = FNV_OFFSET
	local ok = true
	for _ = 1, 120000 do
		local f
		for n = 0, 20 do
			f = factorial(n)
			h = fnv(h, f)
		end
		ok = ok and f == 2432902008176640000
	end
	return report("factorialiter", h, ok)
end

os.exit(main())
