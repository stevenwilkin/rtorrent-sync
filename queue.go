package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

type downloader struct {
	host string // ssh host rtorrent runs on
	dir  string // rtorrent's download directory on host
	dest string // local destination directory
}

// queue adds a download of the named torrent to nq's queue, equivalent to:
//
//	nq rsync -av --progress host:'dir/name' dest
//
// The remote path is passed through ssh to the remote shell, which would
// otherwise interpret spaces, brackets and other special characters, so it is
// single-quoted here. RSYNC_OLD_ARGS stops rsync 3.2.4+ from also protecting
// the path itself, which would leave the quotes in the filename; older rsyncs
// and macOS's openrsync ignore it.
func (d downloader) queue(name string) error {
	src := d.host + ":" + shellQuote(path.Join(d.dir, name))
	cmd := exec.Command("nq", "rsync", "-av", "--progress", src, d.dest)
	cmd.Env = append(os.Environ(), "RSYNC_OLD_ARGS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

// shellQuote quotes s for a POSIX shell, wrapping it in single quotes and
// escaping any single quotes within it.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
