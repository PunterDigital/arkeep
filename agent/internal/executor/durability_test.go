package executor

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/arkeep-io/arkeep/shared/proto"
)

// mountinfoLine builds one /proc/self/mountinfo line.
func mountinfoLine(mountPoint, fsType string) string {
	return "36 35 0:31 / " + mountPoint + " rw,relatime shared:1 - " + fsType + " src rw"
}

func TestParseMountInfo(t *testing.T) {
	data := strings.Join([]string{
		"1180 1061 0:131 / / rw,relatime master:578 - overlay overlay rw,lowerdir=/x,upperdir=/y,workdir=/z",
		"1186 1180 8:1 /srv/my\\040backups /backups rw,relatime - ext4 /dev/sda1 rw",
		"1190 1180 8:1 / /hostfs ro,relatime - ext4 /dev/sda1 ro",
		"1191 1180 0:40 / /dev/shm rw,nosuid,nodev,noexec - tmpfs shm rw,size=65536k",
		"1192 1180 8:1 /data /mnt/with\\040space rw - ext4 /dev/sda1 rw",
		"garbage",
		"",
	}, "\n")
	got := parseMountInfo(data)
	want := []mountEntry{
		{"/", "overlay"},
		{"/backups", "ext4"},
		{"/hostfs", "ext4"},
		{"/dev/shm", "tmpfs"},
		{"/mnt/with space", "ext4"},
	}
	if len(got) != len(want) {
		t.Fatalf("parseMountInfo() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestMountFor(t *testing.T) {
	entries := []mountEntry{
		{"/", "overlay"},
		{"/hostfs", "ext4"},
		{"/hostfs/mnt/user", "xfs"},
		{"/backups", "ext4"},
		{"/backups", "tmpfs"}, // mounted over the one above
	}
	tests := []struct {
		path, wantMount, wantType string
	}{
		{"/backups/repo", "/backups", "tmpfs"},
		{"/backupsX/repo", "/", "overlay"},
		{"/hostfs/mnt/user/Arkeep", "/hostfs/mnt/user", "xfs"},
		{"/hostfs/home/u", "/hostfs", "ext4"},
		{"/var/lib/arkeep", "/", "overlay"},
	}
	for _, tt := range tests {
		m, ok := mountFor(entries, tt.path)
		if !ok || m.mountPoint != tt.wantMount || m.fsType != tt.wantType {
			t.Errorf("mountFor(%q) = %+v, %v; want %s (%s)", tt.path, m, ok, tt.wantMount, tt.wantType)
		}
	}
}

func TestNonDurableLocalPath(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.ToSlash(filepath.Join(dir, "repo"))
	tests := []struct {
		name      string
		mountinfo string
		err       error
		wantMsg   string // substring; "" means durable
	}{
		{
			name:      "container root only (the trial)",
			mountinfo: mountinfoLine("/", "overlay"),
			wantMsg:   "is not on a volume or bind mount",
		},
		{
			name:      "bind mount covers the repository",
			mountinfo: mountinfoLine("/", "overlay") + "\n" + mountinfoLine(filepath.ToSlash(dir), "ext4"),
		},
		{
			name:      "tmpfs covers the repository",
			mountinfo: mountinfoLine("/", "overlay") + "\n" + mountinfoLine(filepath.ToSlash(dir), "tmpfs"),
			wantMsg:   "tmpfs mount",
		},
		{
			name: "mount table unreadable",
			err:  errors.New("no /proc"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(nil, nil, nil, zap.NewNop(), "/hostfs", "")
			e.mountInfo = func() ([]byte, error) { return []byte(tt.mountinfo), tt.err }
			// EvalSymlinks would resolve a symlinked temp dir (macOS /var → /private/var)
			// to a path the fake mount table does not list: compare unresolved.
			got := e.nonDurableLocalPath(repo, "/backups/repo")
			if tt.wantMsg == "" {
				if got != "" {
					t.Fatalf("nonDurableLocalPath() = %q, want durable", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantMsg) || !strings.Contains(got, "/backups/repo") {
				t.Fatalf("nonDurableLocalPath() = %q, want it to contain %q and the path as entered", got, tt.wantMsg)
			}
		})
	}

	t.Run("native agent is never checked", func(t *testing.T) {
		e := New(nil, nil, nil, zap.NewNop(), "", "")
		if e.mountInfo != nil {
			t.Fatal("a native agent (no host root) must not read the mount table")
		}
		if got := e.nonDurableLocalPath(repo, repo); got != "" {
			t.Fatalf("nonDurableLocalPath() = %q, want \"\" for a native agent", got)
		}
	})
}

// A backup to a local destination that is not on any mount inside the agent
// container fails for that destination instead of succeeding into storage
// that disappears with the container; with a mount there, it succeeds.
func TestBackupE2E_NonDurableLocalDestination(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mounted   bool
		wantState string
	}{
		{"container layer", false, "failed"},
		{"bind mount", true, "succeeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newE2EEnv(t, true)
			env.writeSource(t)
			repo := filepath.ToSlash(env.repoPath())
			lines := mountinfoLine("/", "overlay")
			if tc.mounted {
				lines += "\n" + mountinfoLine(filepath.Dir(repo), "ext4")
			}
			env.e.mountInfo = func() ([]byte, error) { return []byte(lines), nil }

			srcJSON, _ := json.Marshal([]string{env.path(e2eSource)})
			raw, err := json.Marshal(backupPayload{
				Sources:      string(srcJSON),
				RepoPassword: "test-password",
				Destinations: []destinationPayload{{DestinationID: "dest-1", Type: "local", RepoURL: env.repoURL}},
			})
			if err != nil {
				t.Fatal(err)
			}
			sink := &recordingSink{}
			reporter := &fakeReporter{}
			env.e.executeBackup(context.Background(), JobAssignment{JobID: "backup-1", Type: proto.JobType_JOB_TYPE_BACKUP, Payload: raw}, sink, reporter)

			if got := lastStatus(reporter); got != tc.wantState {
				t.Fatalf("backup status = %q (%s), want %s; logs:\n%v", got, reporter.lastMsg, tc.wantState, sink.logs)
			}
			if !tc.mounted {
				if !sink.contains("is not on a volume or bind mount") {
					t.Errorf("logs do not explain the failure:\n%v", sink.logs)
				}
				if id := reporter.snapshotIDs["dest-1"]; id != "" {
					t.Errorf("reported snapshot %s for a non-durable destination", id)
				}
			}
		})
	}
}
