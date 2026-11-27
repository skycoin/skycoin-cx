-- Encodes a point and 16 i32 into a 72 byte little endian buffer and back, 76000 times.
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

local buf = {}
for i = 1, 72 do
	buf[i] = 0
end

local function encode(at, v)
	buf[at + 1] = v & 255
	buf[at + 2] = (v >> 8) & 255
	buf[at + 3] = (v >> 16) & 255
	buf[at + 4] = (v >> 24) & 255
end

local function decode(at)
	return buf[at + 1] | buf[at + 2] << 8 | buf[at + 3] << 16 | buf[at + 4] << 24
end

local function main()
	local h = FNV_OFFSET
	local ok = true
	local p = { x = 0, y = 0 }
	local q = { x = 0, y = 0 }
	local input = { ints = {} }
	local output = { ints = {} }
	for round = 0, 75999 do
		p.x = round
		p.y = 2 * round
		for k = 0, 15 do
			input.ints[k + 1] = 16 * round + k
		end
		encode(0, p.x)
		encode(4, p.y)
		for k = 0, 15 do
			encode(8 + 4 * k, input.ints[k + 1])
		end
		q.x = decode(0)
		q.y = decode(4)
		for k = 0, 15 do
			output.ints[k + 1] = decode(8 + 4 * k)
		end
		local sum = q.x + q.y
		for k = 1, 16 do
			sum = sum + output.ints[k]
		end
		h = fnv(h, sum)
		ok = ok and sum == 259 * round + 120
	end
	return report("structserialize", h, ok)
end

os.exit(main())
