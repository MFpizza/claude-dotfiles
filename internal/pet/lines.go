package pet

// Stage is the emoji a pet shows from Level on.
type Stage struct {
	Level int
	Emoji string
}

// Branch is one way a line can grow after it splits.
type Branch struct {
	Key    string
	Stages []Stage
}

// Line is a family of pets: shared early stages, then optionally a random branch.
type Line struct {
	Key      string
	Stages   []Stage
	Branches []Branch
}

// Lines hatch from 🥚 (egg layers), 🌰 (plants) or 🍼 (mammals). Emoji stay within
// Unicode 12: newer ones such as 🦭 render as boxes in many terminals.
var Lines = []Line{
	{"bird", []Stage{{1, "🥚"}, {3, "🐣"}, {5, "🐥"}, {7, "🐔"}}, []Branch{
		{"fowl", []Stage{{10, "🐓"}, {13, "🦃"}, {16, "🦚"}, {20, "👑🦚"}}},
		{"raptor", []Stage{{10, "🐦"}, {13, "🦉"}, {16, "🦅"}, {20, "👑🦅"}}},
	}},
	{"water", []Stage{{1, "🥚"}, {3, "🐣"}, {5, "🐥"}, {7, "🦆"}, {10, "🐧"}, {13, "🦩"}, {16, "🦢"}, {20, "👑🦢"}}, nil},
	{"reptile", []Stage{{1, "🥚"}, {3, "🦎"}, {5, "🐢"}, {7, "🐍"}, {10, "🐊"}}, []Branch{
		{"dino", []Stage{{13, "🦕"}, {16, "🦖"}, {20, "👑🦖"}}},
		{"dragon", []Stage{{13, "🐲"}, {16, "🐉"}, {20, "👑🐉"}}},
	}},
	{"sea", []Stage{{1, "🥚"}, {3, "🦐"}, {5, "🐟"}, {7, "🐠"}, {10, "🐡"}}, []Branch{
		{"cephalopod", []Stage{{13, "🦑"}, {16, "🐙"}, {20, "👑🐙"}}},
		{"crustacean", []Stage{{13, "🦀"}, {16, "🦞"}, {20, "👑🦞"}}},
	}},
	{"insect", []Stage{{1, "🥚"}, {3, "🐛"}, {5, "🐜"}, {7, "🐞"}, {10, "🦗"}, {13, "🐝"}, {16, "🦋"}, {20, "👑🦋"}}, nil},
	{"plant", []Stage{{1, "🌰"}, {3, "🌱"}, {5, "🌿"}, {7, "🍀"}}, []Branch{
		{"flower", []Stage{{10, "🌷"}, {13, "🌹"}, {16, "🌻"}, {20, "👑🌻"}}},
		{"sakura", []Stage{{10, "🌳"}, {13, "🌸"}, {16, "🍒"}, {20, "👑🌸"}}},
		{"apple", []Stage{{10, "🌳"}, {13, "🍏"}, {16, "🍎"}, {20, "👑🍎"}}},
		{"grape", []Stage{{10, "🍃"}, {13, "🍇"}, {16, "🍷"}, {20, "👑🍷"}}},
	}},
	{"mammal", []Stage{{1, "🍼"}, {3, "🐾"}}, []Branch{
		{"feline", []Stage{{5, "🐱"}, {7, "🐈"}, {10, "🐆"}, {13, "🐅"}, {16, "🦁"}, {20, "👑🦁"}}},
		{"canine", []Stage{{5, "🐶"}, {10, "🐕"}, {16, "🐺"}, {20, "👑🐺"}}},
		{"marine", []Stage{{5, "🦦"}, {10, "🐬"}, {13, "🐳"}, {16, "🐋"}, {20, "👑🐋"}}},
		{"primate", []Stage{{5, "🐵"}, {7, "🙈"}, {10, "🐒"}, {13, "🦧"}, {16, "🦍"}, {20, "👑🦍"}}},
	}},
}
