package storage

import (
	"hash/fnv"
	"strings"
)

// Палитра дефолтных цветов — те же, что на клиенте.
var defaultColors = []string{
	"#e57373", "#ba68c8", "#7986cb", "#4fc3f7", "#4db6ac",
	"#81c784", "#ffd54f", "#ffb74d", "#a1887f", "#90a4ae",
}

func DefaultColorFor(name string) string {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(name)))
	return defaultColors[int(h.Sum32())%len(defaultColors)]
}
