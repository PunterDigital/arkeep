package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// readSelfMountInfo returns the agent's own mount table.
func readSelfMountInfo() ([]byte, error) {
	return os.ReadFile("/proc/self/mountinfo")
}

// mountEntry is one line of /proc/self/mountinfo: where something is mounted
// and its filesystem type.
type mountEntry struct {
	mountPoint string
	fsType     string
}

// parseMountInfo reads the mount point (field 5) and filesystem type (the
// first field after the "-" separator) of each line of /proc/self/mountinfo.
// Malformed lines are skipped.
func parseMountInfo(data string) []mountEntry {
	var entries []mountEntry
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) < 7 {
			continue
		}
		sep := -1
		for i := 6; i < len(f); i++ {
			if f[i] == "-" {
				sep = i
				break
			}
		}
		if sep < 0 || sep+1 >= len(f) {
			continue
		}
		entries = append(entries, mountEntry{mountPoint: unescapeMountInfo(f[4]), fsType: f[sep+1]})
	}
	return entries
}

// unescapeMountInfo decodes the octal escapes (\040 for a space, \011, \012,
// \134) the kernel writes in mountinfo paths.
func unescapeMountInfo(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+4 <= len(s) {
			if v, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// mountFor returns the mount that holds p: the entry with the longest mount
// point that is p or one of its ancestors. A later entry for the same mount
// point wins, since it is mounted over the earlier one.
func mountFor(entries []mountEntry, p string) (mountEntry, bool) {
	var best mountEntry
	found := false
	for _, m := range entries {
		if !pathWithin(p, m.mountPoint) {
			continue
		}
		if !found || len(m.mountPoint) >= len(best.mountPoint) {
			best, found = m, true
		}
	}
	return best, found
}

// volatileFSTypes are filesystems whose contents are gone after a restart.
var volatileFSTypes = map[string]bool{"tmpfs": true, "ramfs": true}

// nonDurableLocalPath returns an error message when the local destination
// repoPath (already translated to the agent's view) would not outlive the
// agent's container: it is not on any bind mount or volume, so it lives in
// the container's own writable layer, or it is on a tmpfs. Such a backup
// succeeds and is lost when the container is recreated. It returns "" when
// the path is durable, for a native agent (no mountInfo), and whenever the
// mount table cannot be read, so the check never blocks a backup on its own
// failure. originalURL is the path as the user entered it, for the message.
func (e *Executor) nonDurableLocalPath(repoPath, originalURL string) string {
	if e.mountInfo == nil {
		return ""
	}
	data, err := e.mountInfo()
	if err != nil {
		return ""
	}
	resolved := filepath.Clean(repoPath)
	if r, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = r
	}
	m, ok := mountFor(parseMountInfo(string(data)), filepath.ToSlash(resolved))
	if !ok {
		return ""
	}
	switch {
	case m.mountPoint == "/":
		return fmt.Sprintf(
			"local path %q (%q inside the agent container) is not on a volume or bind mount: it is stored in the agent "+
				"container's own filesystem and would be lost when the container is recreated — mount a host directory "+
				"or a volume at that path (e.g. -v /srv/backups:%s), or use a path under the host-root mount (%s)",
			originalURL, resolved, resolved, e.dockerHostRoot)
	case volatileFSTypes[m.fsType]:
		return fmt.Sprintf(
			"local path %q (%q inside the agent container) is on a %s mount at %q, which is emptied when the container "+
				"restarts — mount a host directory or a volume there instead",
			originalURL, resolved, m.fsType, m.mountPoint)
	}
	return ""
}
