package history

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// adjectives is a list of descriptive words for memorable ID generation.
var adjectives = []string{
	"able", "active", "agile", "alert", "alive",
	"amber", "ample", "apt", "aqua", "arch",
	"avid", "azure", "basic", "bliss", "bold",
	"brave", "brief", "bright", "brisk", "broad",
	"calm", "candid", "chief", "civic", "civil",
	"clean", "clear", "clever", "close", "cobalt",
	"cool", "coral", "core", "cosmic", "crisp",
	"cyan", "daring", "deft", "dense", "direct",
	"dual", "eager", "early", "easy", "elder",
	"epic", "equal", "exact", "extra", "fair",
	"fast", "fawn", "fine", "firm", "first",
	"fit", "fleet", "focal", "fond", "frank",
	"free", "fresh", "front", "full", "game",
	"gentle", "giant", "glad", "global", "gold",
	"good", "grand", "great", "green", "hale",
	"handy", "happy", "hardy", "hazel", "hearty",
	"heavy", "hep", "high", "honey", "ideal",
	"inner", "jade", "jolly", "jovial", "just",
	"keen", "key", "kind", "known", "laser",
	"late", "lead", "lean", "level", "light",
	"lithe", "live", "lively", "local", "logic",
	"lone", "long", "loud", "loyal", "lucid",
	"lunar", "lush", "lyric", "magic", "main",
	"major", "maple", "master", "mellow", "merry",
	"metal", "micro", "mighty", "mint", "model",
	"moral", "native", "naval", "near", "neat",
	"new", "next", "nimble", "noble", "north",
	"novel", "oaken", "olive", "open", "optimal",
	"outer", "patient", "peak", "pearl", "pilot",
	"pine", "plain", "plum", "polar", "polite",
	"primal", "prime", "prior", "proud", "pure",
	"quick", "quiet", "radiant", "rapid", "rare",
	"real", "ready", "regal", "rich", "right",
	"robust", "rosy", "royal", "ruby", "rural",
	"rustic", "safe", "sage", "satin", "savvy",
	"sharp", "sheer", "sleek", "slick", "smart",
	"smooth", "snug", "solar", "solid", "sonic",
	"sound", "south", "spare", "spry", "stable",
	"stark", "steady", "steel", "stellar", "still",
	"stout", "strong", "sturdy", "subtle", "super",
}

// nouns is a list of concrete nouns for memorable ID generation.
var nouns = []string{
	"alder", "amber", "anchor", "apex", "arch",
	"arrow", "aspen", "atlas", "aurora", "badge",
	"basin", "beacon", "beam", "birch", "blade",
	"blaze", "bloom", "bolt", "bower", "branch",
	"brass", "breeze", "brick", "bridge", "brook",
	"cairn", "canyon", "cape", "cedar", "charm",
	"chime", "chrome", "cinder", "circuit", "citrus",
	"clay", "cliff", "cloud", "clover", "coast",
	"comet", "copper", "coral", "cosmos", "cove",
	"craft", "crane", "crest", "creek", "crown",
	"crystal", "current", "cypress", "dawn", "delta",
	"dew", "diamond", "drift", "dune", "dust",
	"eagle", "earth", "echo", "edge", "elm",
	"ember", "emerald", "falcon", "fawn", "feather",
	"fern", "field", "finch", "fire", "fjord",
	"flame", "flare", "flint", "flora", "forge",
	"fossil", "frost", "gale", "gate", "gem",
	"geyser", "glacier", "glade", "glass", "glen",
	"globe", "gold", "gorge", "grain", "granite",
	"grove", "gust", "harbor", "haven", "hawk",
	"hazel", "heath", "hedge", "helm", "heron",
	"hill", "hollow", "horizon", "horn", "husk",
	"icon", "inlet", "iris", "iron", "isle",
	"ivory", "ivy", "jade", "jasper", "jet",
	"jewel", "jungle", "juniper", "keel", "kelp",
	"kernel", "key", "knoll", "lake", "lantern",
	"larch", "lark", "lava", "lawn", "leaf",
	"ledge", "light", "lily", "linden", "lodge",
	"lotus", "lunar", "lynx", "maple", "marble",
	"marsh", "meadow", "mesa", "metal", "mist",
	"moon", "moss", "mound", "mount", "nectar",
	"nest", "node", "north", "nova", "oak",
	"oasis", "ocean", "olive", "onyx", "orbit",
	"orchid", "ore", "osprey", "otter", "palm",
	"pass", "path", "peak", "pearl", "pebble",
	"pier", "pine", "pixel", "plain", "plum",
	"point", "pond", "poplar", "port", "prairie",
	"prism", "pulse", "quartz", "rain", "range",
	"rapid", "raven", "ray", "reef", "ridge",
	"river", "robin", "rock", "root", "rose",
	"rust", "sage", "sand", "sapphire", "scale",
	"seed", "shade", "shell", "shore", "shrub",
	"silver", "slate", "snow", "solar", "sonic",
	"spark", "spire", "spring", "spruce", "spur",
	"star", "steam", "steel", "stem", "stone",
	"storm", "strait", "stream", "summit", "sun",
	"surge", "swan", "swift", "terra", "thicket",
	"thorn", "thunder", "tide", "timber", "torch",
	"tower", "trail", "trench", "tundra", "vale",
	"valley", "vapor", "vault", "velvet", "verge",
	"vine", "void", "vortex", "wave", "willow",
	"wind", "wing", "winter", "wood", "wren",
	"zenith", "zephyr", "zinc", "zone", "zest",
}

// GenerateID creates a unique identifier in adjective_noun_YYYYMMDD_HHMMSS format.
// Uses crypto/rand for secure random word selection to prevent collisions.
func GenerateID() (string, error) {
	adj, err := randomWord(adjectives)
	if err != nil {
		return "", fmt.Errorf("selecting random adjective: %w", err)
	}

	noun, err := randomWord(nouns)
	if err != nil {
		return "", fmt.Errorf("selecting random noun: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s_%s", adj, noun, timestamp), nil
}

// randomWord selects a random word from the given slice using crypto/rand.
func randomWord(words []string) (string, error) {
	if len(words) == 0 {
		return "", fmt.Errorf("word list is empty")
	}

	max := big.NewInt(int64(len(words)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generating random number: %w", err)
	}

	return words[n.Int64()], nil
}
