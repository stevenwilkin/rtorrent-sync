# rtorrent-sync

A terminal UI for pulling completed downloads from a remote
[rtorrent](https://github.com/rakshasa/rtorrent) server to the local machine.

It talks to rtorrent's XML-RPC API over SCGI, through an SSH-forwarded Unix
socket, and lists the completed torrents. Selecting a torrent queues an
`rsync` of it from the server using [nq](https://github.com/leahneukirchen/nq),
so downloads run one at a time in the background and carry on after the app
exits.

`rtorrent-sync.sh` sets up the port forwarding, runs the app and tears the
forwarding down again on exit.

## Enable XML-RPC over Unix domain socket

### Update ~/.rtorrent.rc

	scgi_local = /tmp/rtorrent.sock

### Create socket file

As root:

	touch /tmp/rtorrent.sock
	chown USER /tmp/rtorrent.sock

## Forward local port to remote Unix socket

	ssh -N -L 1234:/tmp/rtorrent.sock HOST
