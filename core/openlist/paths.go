package openlist

import (
	stdpath "path"
	"strings"
)

func BuildOpenListPath(songPath, libraryRoot string) string {
	path := strings.ReplaceAll(strings.TrimSpace(songPath), "\\", "/")
	if path == "" {
		return ""
	}
	root := strings.TrimSpace(libraryRoot)
	if root == "" {
		return ""
	}
	root = ensureLeadingSlash(strings.TrimRight(strings.ReplaceAll(root, "\\", "/"), "/"))

	rel := path
	if strings.HasPrefix(rel, root+"/") {
		rel = strings.TrimPrefix(rel, root+"/")
	} else if rel == root {
		rel = ""
	}
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return root
	}
	return root + "/" + rel
}

func BuildCoverPath(songPath string) string {
	clean := strings.ReplaceAll(strings.TrimSpace(songPath), "\\", "/")
	if clean == "" {
		return ""
	}
	dir := stdpath.Dir(clean)
	if dir == "." {
		dir = ""
	}
	if dir == "/" {
		return "/cover.jpg"
	}
	if dir == "" {
		return "cover.jpg"
	}
	return stdpath.Join(dir, "cover.jpg")
}

func normalizeBase(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

func ensureLeadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func resolveRawURL(rawURL, base string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	base = normalizeBase(base)
	if strings.HasPrefix(rawURL, "/") {
		return base + rawURL
	}
	return base + "/" + rawURL
}
