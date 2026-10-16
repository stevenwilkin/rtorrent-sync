package main

import (
	"fmt"
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
// The name is passed as a single argument; rsync (3.2.4+) protects it from the
// remote shell, so spaces and special characters need no quoting.
func (d downloader) queue(name string) error {
	src := d.host + ":" + path.Join(d.dir, name)
	out, err := exec.Command("nq", "rsync", "-av", "--progress", src, d.dest).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}
