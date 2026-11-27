-- The digital root and additive persistence of every n below 1,200,000.
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

local function digitSum(n)
	local sum = 0
	while n > 0 do
		sum = sum + n % 10
		n = n // 10
	end
	return sum
end

local function main()
	local h = FNV_OFFSET
	local ok = true
	for n = 1, 1199999 do
		local root = n
		local persistence = 0
		while root >= 10 do
			root = digitSum(root)
			persistence = persistence + 1
		end
		h = fnv(h, root + 10 * persistence)
		ok = ok and root == 1 + (n - 1) % 9
	end
	return report("digitalroot", h, ok)
end

os.exit(main())
