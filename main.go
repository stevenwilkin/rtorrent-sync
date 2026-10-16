package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"os"
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

func main() {
	addr := flag.String("addr", "localhost:1234", "rtorrent SCGI address")
	flag.Parse()

	res, err := call(*addr, "d.multicall2", "", "complete", "d.name=")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if res.Array == nil {
		fmt.Fprintln(os.Stderr, "unexpected response: not an array")
		os.Exit(1)
	}

	for _, row := range *res.Array {
		if row.Array == nil || len(*row.Array) == 0 {
			continue
		}
		fields := *row.Array
		fmt.Printf("%s\n", fields[0].str())
	}
}
