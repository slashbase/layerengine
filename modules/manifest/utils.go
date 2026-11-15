package manifest

// The rules for names that end up in generated Lua. Two kinds of name have to
// satisfy them: a module name, which becomes a global in the layer sandbox, and
// a manifest function name, which becomes a path of table lookups below that
// global. Both are rendered into code the model then writes, so both are held
// to the identifier pattern and the reserved-word list.
//
// They live here rather than in a package of their own because this package is
// the leaf that already owns the manifest vocabulary, and package modules
// already imports it for the types. A separate package would add a dependency
// edge without removing one.

// IsLuaIdentifier reports whether s matches [A-Za-z_][A-Za-z0-9_]*.
func IsLuaIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
			// allowed anywhere in the segment
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// IsLuaKeyword reports whether s is a Lua 5.1 reserved word: it matches the
// identifier pattern but cannot be used as a name, so <module>.end(...) would
// not parse.
func IsLuaKeyword(s string) bool {
	return luaKeywords[s]
}

// luaKeywords are the Lua 5.1 reserved words.
var luaKeywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true,
	"end": true, "false": true, "for": true, "function": true, "if": true,
	"in": true, "local": true, "nil": true, "not": true, "or": true,
	"repeat": true, "return": true, "then": true, "true": true, "until": true,
	"while": true,
}
