package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

type value struct {
	String *string  `xml:"string"`
	Int    *string  `xml:"int"`
	I4     *string  `xml:"i4"`
	I8     *string  `xml:"i8"`
	Array  *[]value `xml:"array>data>value"`
	Chars  string   `xml:",chardata"`
}

type methodResponse struct {
	Params []value `xml:"params>param>value"`
	Fault  *struct {
		Members []struct {
			Name  string `xml:"name"`
			Value value  `xml:"value"`
		} `xml:"value>struct>member"`
	} `xml:"fault"`
}

func (v value) str() string {
	switch {
	case v.String != nil:
		return *v.String
	case v.Int != nil:
		return *v.Int
	case v.I4 != nil:
		return *v.I4
	case v.I8 != nil:
		return *v.I8
	}
	// A <value> with no type element is a string.
	return v.Chars
}

// call performs an XML-RPC call over SCGI and returns the first result param.
func call(addr, method string, params ...string) (value, error) {
	var body bytes.Buffer
	body.WriteString(`<?xml version="1.0"?><methodCall><methodName>`)
	xml.EscapeText(&body, []byte(method))
	body.WriteString(`</methodName><params>`)
	for _, p := range params {
		body.WriteString(`<param><value><string>`)
		xml.EscapeText(&body, []byte(p))
		body.WriteString(`</string></value></param>`)
	}
	body.WriteString(`</params></methodCall>`)

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return value{}, err
	}
	defer conn.Close()

	// SCGI request: netstring-encoded headers followed by the body.
	headers := "CONTENT_LENGTH\x00" + strconv.Itoa(body.Len()) + "\x00SCGI\x001\x00"
	req := strconv.Itoa(len(headers)) + ":" + headers + "," + body.String()
	if _, err := io.WriteString(conn, req); err != nil {
		return value{}, err
	}

	// SCGI response: CGI-style headers, blank line, then the XML body.
	r := bufio.NewReader(conn)
	if _, err := textproto.NewReader(r).ReadMIMEHeader(); err != nil {
		return value{}, fmt.Errorf("reading response headers: %w", err)
	}

	var resp methodResponse
	if err := xml.NewDecoder(r).Decode(&resp); err != nil {
		return value{}, fmt.Errorf("decoding response: %w", err)
	}
	if resp.Fault != nil {
		var parts []string
		for _, m := range resp.Fault.Members {
			parts = append(parts, m.Name+"="+m.Value.str())
		}
		return value{}, fmt.Errorf("fault: %s", strings.Join(parts, ", "))
	}
	if len(resp.Params) == 0 {
		return value{}, fmt.Errorf("empty response")
	}
	return resp.Params[0], nil
}

// completedTorrents returns the names of all torrents in rtorrent's "complete" view.
func completedTorrents(addr string) ([]string, error) {
	res, err := call(addr, "d.multicall2", "", "complete", "d.name=")
	if err != nil {
		return nil, err
	}
	if res.Array == nil {
		return nil, fmt.Errorf("unexpected response: not an array")
	}

	var names []string
	for _, row := range *res.Array {
		if row.Array == nil || len(*row.Array) == 0 {
			continue
		}
		names = append(names, (*row.Array)[0].str())
	}
	return names, nil
}
