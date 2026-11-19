package rpg

import (
	"strconv"
	"strings"
)

// noteValue returns the text after "key=" on the first matching line.
func noteValue(note, key string) (string, bool) {
	prefix := key + "="
	for _, line := range strings.Split(note, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return "", false
}

// NoteNumber reads an integer note tag, returning 0 when absent.
func NoteNumber(note, key string) int {
	v, ok := noteValue(note, key)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	return n
}

// NoteNumberPresent reads an integer note tag and reports whether it exists.
func NoteNumberPresent(note, key string) (int, bool) {
	v, ok := noteValue(note, key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, true
	}
	return n, true
}

// NoteBool reads a boolean note tag. Only the exact word "true" (any case)
// enables it, matching the original Note.read_boolean semantics.
func NoteBool(note, key string) bool {
	v, ok := noteValue(note, key)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(v), "true")
}

// NoteString reads a string note tag.
func NoteString(note, key string) string {
	v, _ := noteValue(note, key)
	return v
}

// ParseSpriteRefs parses "Name/Index,Name/Index". A missing index is 0.
func ParseSpriteRefs(value string) []SpriteRef {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]SpriteRef, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name := part
		index := 0
		if slash := strings.LastIndex(part, "/"); slash >= 0 {
			name = strings.TrimSpace(part[:slash])
			if n, err := strconv.Atoi(strings.TrimSpace(part[slash+1:])); err == nil {
				index = n
			}
		}
		if name == "" {
			continue
		}
		out = append(out, SpriteRef{Name: name, Index: index})
	}
	return out
}

// ParseClassGraphics reads MaleSprites, FemaleSprites, MaleFaces and
// FemaleFaces from a class note.
func ParseClassGraphics(note string) ClassGraphics {
	return ClassGraphics{
		MaleSprites:   ParseSpriteRefs(NoteString(note, "MaleSprites")),
		FemaleSprites: ParseSpriteRefs(NoteString(note, "FemaleSprites")),
		MaleFaces:     ParseSpriteRefs(NoteString(note, "MaleFaces")),
		FemaleFaces:   ParseSpriteRefs(NoteString(note, "FemaleFaces")),
	}
}

// ParsePaperdoll reads "Paperdoll=Name/Index".
func ParsePaperdoll(note string) (string, int) {
	refs := ParseSpriteRefs(NoteString(note, "Paperdoll"))
	if len(refs) == 0 {
		return "", 0
	}
	return refs[0].Name, refs[0].Index
}
