# rtorrent-sync

## Enable XML-RPC over Unix domain socket

### Update ~/.rtorrent.rc

	scgi_local = /tmp/rtorrent.sock

### Create socket file

As root:

	touch /tmp/rtorrent.sock
	chown USER /tmp/rtorrent.sock

## Forward local port to remote Unix socket

	ssh -N -L 1234:/tmp/rtorrent.sock HOST
