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
	titleStyle       = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	cursorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	highlightedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("214"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

type torrentsMsg []string
type errMsg error

type model struct {
	addr        string
	torrents    []string
	highlighted map[int]bool
	cursor      int
	offset      int // index of the first visible row
	width       int // terminal width
	height      int // terminal height
	loading     bool
	err         error
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

	case tea.KeyMsg:
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
			if len(m.torrents) > 0 {
				m.highlighted[m.cursor] = !m.highlighted[m.cursor]
			}
		case "r":
			m.loading = true
			m.err = nil
			m.highlighted = map[int]bool{}
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

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
	}

	end := min(m.offset+m.visibleRows(), len(m.torrents))
	for i := m.offset; i < end; i++ {
		name := m.torrents[i]
		if m.width > 2 {
			// Truncate so long names don't wrap and throw off scrolling.
			name = ansi.Truncate(name, m.width-2, "…")
		}
		if m.highlighted[i] {
			name = highlightedStyle.Render(name)
		}
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("> ") + name + "\n")
		} else {
			b.WriteString("  " + name + "\n")
		}
	}

	b.WriteString(helpStyle.Render("↑/↓ move • enter highlight • r refresh • q quit"))
	return b.String()
}

func main() {
	addr := flag.String("addr", "localhost:1234", "rtorrent SCGI address")
	flag.Parse()

	m := model{addr: *addr, highlighted: map[int]bool{}, loading: true}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
