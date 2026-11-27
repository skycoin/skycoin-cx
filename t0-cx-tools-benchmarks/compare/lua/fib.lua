-- The recursive fib(34), 18,454,929 calls.
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

local FIB_N = 34
local FIB_WANT = 5702887

local function fib(n)
	if n < 2 then
		return n
	end
	return fib(n - 1) + fib(n - 2)
end

local function main()
	local r = fib(FIB_N)
	return report("fib", fnv(FNV_OFFSET, r), r == FIB_WANT)
end

os.exit(main())
