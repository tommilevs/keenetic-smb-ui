// Package acl resolves per-folder access rules with Keenetic-style inheritance.
package acl

import (
	"path"
	"strings"
)

// Access is the effective permission of a user on a folder.
type Access string

const (
	// None means the user must not see the folder at all.
	None Access = "none"
	// ReadOnly allows listing and reading.
	ReadOnly Access = "ro"
	// ReadWrite allows writing too.
	ReadWrite Access = "rw"
)

// Rule is a single user entry attached to a folder.
type Rule struct {
	User    string `json:"user"`
	Access  string `json:"access"`  // "rw" | "ro" | "none"
	Inherit bool   `json:"inherit"` // propagate the rule to subfolders
}

// Folder is a directory inside the exported share with its own access list.
type Folder struct {
	ID      string `json:"id"`
	Path    string `json:"path"` // relative to the share root, "" = root
	Name    string `json:"name"` // Samba share name (auto-derived when empty)
	Enabled bool   `json:"enabled"`
	Rules   []Rule `json:"rules"`
}

// CleanPath normalises a folder path to the "a/b" form (no leading slash).
func CleanPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	return path.Clean(p)
}

// ShareName returns the SMB share name for a folder: explicit name if set,
// otherwise the last path element prefixed with "folder-".
func ShareName(f Folder) string {
	if strings.TrimSpace(f.Name) != "" {
		return strings.TrimSpace(f.Name)
	}
	p := CleanPath(f.Path)
	if p == "" {
		return "root"
	}
	base := path.Base(p)
	return "folder-" + strings.ReplaceAll(base, " ", "_")
}

// Effective resolves the permission of user on folder, walking from the folder
// itself to the root. A rule found on the folder always applies; a rule found on
// an ancestor applies only when that rule has Inherit enabled.
func Effective(folders []Folder, folderPath, user string) Access {
	target := CleanPath(folderPath)
	user = strings.TrimSpace(user)
	if user == "" {
		return None
	}

	// exact folder first
	if f, ok := find(folders, target); ok {
		if rule, ok := findRule(f, user); ok {
			return normalize(rule.Access)
		}
	}

	// then ancestors, nearest first
	for cur := parentPath(target); ; cur = parentPath(cur) {
		if f, ok := find(folders, cur); ok {
			if rule, ok := findRule(f, user); ok && rule.Inherit {
				return normalize(rule.Access)
			}
		}
		if cur == "" {
			break
		}
	}
	return None
}

// InheritedBy reports the folder that provides the inherited rule, if any.
// It is used by the UI to show "inherited from …" hints.
func InheritedBy(folders []Folder, folderPath, user string) (string, bool) {
	target := CleanPath(folderPath)
	for cur := parentPath(target); ; cur = parentPath(cur) {
		if f, ok := find(folders, cur); ok {
			if rule, ok := findRule(f, user); ok && rule.Inherit {
				return cur, true
			}
		}
		if cur == "" {
			break
		}
	}
	return "", false
}

// Writers returns the users with ReadWrite access on folder.
func Writers(folders []Folder, folderPath string, users []string) []string {
	return collect(folders, folderPath, users, ReadWrite)
}

// Readers returns every user who can at least read folder.
func Readers(folders []Folder, folderPath string, users []string) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if a := Effective(folders, folderPath, u); a != None {
			out = append(out, u)
		}
	}
	return out
}

func collect(folders []Folder, folderPath string, users []string, want Access) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if Effective(folders, folderPath, u) == want {
			out = append(out, u)
		}
	}
	return out
}

func find(folders []Folder, p string) (Folder, bool) {
	p = CleanPath(p)
	for _, f := range folders {
		if CleanPath(f.Path) == p {
			return f, true
		}
	}
	return Folder{}, false
}

func findRule(f Folder, user string) (Rule, bool) {
	for _, r := range f.Rules {
		if strings.EqualFold(strings.TrimSpace(r.User), user) {
			return r, true
		}
	}
	return Rule{}, false
}

func parentPath(p string) string {
	p = CleanPath(p)
	if p == "" {
		return ""
	}
	if i := strings.LastIndex(p, "/"); i > 0 {
		return p[:i]
	}
	return ""
}

func normalize(a string) Access {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "rw", "readwrite", "write":
		return ReadWrite
	case "ro", "read", "readonly":
		return ReadOnly
	default:
		return None
	}
}
