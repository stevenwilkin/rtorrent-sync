package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	queuedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("214"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).MarginTop(1)
)

type torrentsMsg []string
type errMsg error

// queuedMsg reports the result of queueing a download.
type queuedMsg struct {
	name string
	err  error
}

type model struct {
	addr       string
	downloader downloader
	torrents   []string
	queued     map[string]bool // keyed by name so it survives a refresh
	cursor     int
	offset     int // index of the first visible row
	width      int // terminal width
	height     int // terminal height
	loading    bool
	status     string // last action, shown in place of the help line
	err        error
}

func fetchTorrents(addr string) tea.Cmd {
	return func() tea.Msg {
		names, err := completedTorrents(addr)
		if err != nil {
			return errMsg(err)
		}
		return torrentsMsg(names)
	}
}

func queueDownload(d downloader, name string) tea.Cmd {
	return func() tea.Msg {
		return queuedMsg{name: name, err: d.queue(name)}
	}
}

func (m model) Init() tea.Cmd {
	return fetchTorrents(m.addr)
}

// visibleRows is how many list rows fit between the title and help lines.
func (m model) visibleRows() int {
	return max(m.height-4, 1)
}

func (m *model) moveCursor(delta int) {
	if len(m.torrents) == 0 {
		return
	}
	m.cursor = min(max(m.cursor+delta, 0), len(m.torrents)-1)
	m.clampOffset()
}

// clampOffset scrolls the view so the cursor stays on screen.
func (m *model) clampOffset() {
	rows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = max(min(m.offset, len(m.torrents)-rows), 0)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffset()

	case torrentsMsg:
		m.loading = false
		m.torrents = msg
		m.cursor = min(m.cursor, max(len(m.torrents)-1, 0))
		m.clampOffset()

	case errMsg:
		m.loading = false
		m.err = msg

	case queuedMsg:
		if msg.err != nil {
			delete(m.queued, msg.name)
			m.err = fmt.Errorf("queueing %s: %w", msg.name, msg.err)
		} else {
			m.status = "Queued " + msg.name
		}

	case tea.KeyMsg:
		m.status = ""
		m.err = nil
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "pgup", "ctrl+b":
			m.moveCursor(-m.visibleRows())
		case "pgdown", "ctrl+f", " ":
			m.moveCursor(m.visibleRows())
		case "home", "g":
			m.moveCursor(-len(m.torrents))
		case "end", "G":
			m.moveCursor(len(m.torrents))
		case "enter":
			if len(m.torrents) == 0 {
				break
			}
			name := m.torrents[m.cursor]
			if m.queued[name] {
				m.status = "Already queued " + name
				break
			}
			m.queued[name] = true
			return m, queueDownload(m.downloader, name)
		case "r":
			m.loading = true
			return m, fetchTorrents(m.addr)
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Completed torrents (%d)", len(m.torrents))
	if m.loading {
		title = "Loading torrents…"
	}
	b.WriteString(titleStyle.Render(title) + "\n")

	end := min(m.offset+m.visibleRows(), len(m.torrents))
	for i := m.offset; i < end; i++ {
		name := m.torrents[i]
		if m.width > 2 {
			// Truncate so long names don't wrap and throw off scrolling.
			name = ansi.Truncate(name, m.width-2, "…")
		}
		if m.queued[m.torrents[i]] {
			name = queuedStyle.Render(name)
		}
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("> ") + name + "\n")
		} else {
			b.WriteString("  " + name + "\n")
		}
	}

	footer, style := "↑/↓ move • enter queue download • r refresh • q quit", helpStyle
	if m.err != nil {
		footer, style = "Error: "+m.err.Error(), errorStyle
	} else if m.status != "" {
		footer = m.status
	}
	if m.width > 0 {
		footer = ansi.Truncate(footer, m.width, "…")
	}
	b.WriteString(style.Render(footer))
	return b.String()
}

func main() {
	addr := flag.String("addr", "localhost:1234", "rtorrent SCGI address")
	var d downloader
	flag.StringVar(&d.host, "host", "btc", "ssh host rtorrent runs on")
	flag.StringVar(&d.dir, "dir", "torrents", "rtorrent download directory on host")
	flag.StringVar(&d.dest, "dest", ".", "local download directory")
	flag.Parse()

	m := model{addr: *addr, downloader: d, queued: map[string]bool{}, loading: true}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
