package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	client  *Client
	input   string
	cursor  int
	output  string
	quitting bool
}

func initialModel() model {
	return model{
		client: NewClient("http://localhost:3678"),
	}
}

type tickMsg struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			cmd := strings.TrimSpace(m.input)
			m.input = ""
			m.cursor = 0
			if cmd == "" {
				return m, nil
			}
			m.output = m.execute(cmd)
			if m.quitting {
				return m, tea.Quit
			}
			return m, nil
		case tea.KeyBackspace:
			if m.cursor > 0 {
				m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
				m.cursor--
			}
		case tea.KeyLeft:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyRight:
			if m.cursor < len(m.input) {
				m.cursor++
			}
		case tea.KeyCtrlA:
			m.cursor = 0
		case tea.KeyCtrlE:
			m.cursor = len(m.input)
		case tea.KeyCtrlU:
			m.input = m.input[m.cursor:]
			m.cursor = 0
		case tea.KeyCtrlK:
			m.input = m.input[:m.cursor]
		default:
			if msg.Type == tea.KeyRunes {
				ch := string(msg.Runes)
				m.input = m.input[:m.cursor] + ch + m.input[m.cursor:]
				m.cursor += len(ch)
			}
		}
	}
	return m, nil
}

func (m *model) execute(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "status", "s":
		s, err := m.client.Status()
		if err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return FormatStatus(s)

	case "play":
		if len(args) == 0 {
			return errorStyle.Render("Usage: play <spotify-uri>")
		}
		if err := m.client.Play(args[0]); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return playStyle.Render("▶ Playing")

	case "pause", "pp":
		if err := m.client.PlayPause(); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return "⏯ Toggled play/pause"

	case "next", "n":
		if err := m.client.Next(); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return "⏭ Next track"

	case "prev", "p":
		if err := m.client.Prev(); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return "⏮ Previous track"

	case "vol":
		if len(args) == 0 {
			return errorStyle.Render("Usage: vol <0-100>")
		}
		v, err := strconv.Atoi(args[0])
		if err != nil || v < 0 || v > 100 {
			return errorStyle.Render("Volume must be 0-100")
		}
		if err := m.client.Volume(v); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return fmt.Sprintf("🔊 Volume: %d%%", v)

	case "seek":
		if len(args) == 0 {
			return errorStyle.Render("Usage: seek <seconds>")
		}
		secs, err := strconv.Atoi(args[0])
		if err != nil || secs < 0 {
			return errorStyle.Render("Seek position must be a positive number of seconds")
		}
		if err := m.client.Seek(secs * 1000); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return fmt.Sprintf("⏩ Seeked to %s", FormatTime(secs*1000))

	case "shuffle":
		s, err := m.client.Status()
		if err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		if err := m.client.Shuffle(!s.ShuffleContext); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		if s.ShuffleContext {
			return "🔀 Shuffle off"
		}
		return "🔀 Shuffle on"

	case "queue":
		if len(args) == 0 {
			return errorStyle.Render("Usage: queue <spotify-uri>")
		}
		if err := m.client.Queue(args[0]); err != nil {
			return errorStyle.Render("Error: " + err.Error())
		}
		return "📋 Added to queue"

	case "help", "h":
		return helpText()

	case "quit", "q":
		m.quitting = true
		return "Bye!"

	default:
		return errorStyle.Render(fmt.Sprintf("Unknown command: %s (type 'help' for commands)", cmd))
	}
}

func helpText() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render("spotify-jukebox commands")
	cmds := `
  status, s        Show current track
  play <uri>       Play a Spotify URI
  pause, pp        Toggle play/pause
  next, n          Next track
  prev, p          Previous track
  vol <0-100>      Set volume
  seek <seconds>   Seek to position
  shuffle          Toggle shuffle
  queue <uri>      Add track to queue
  help, h          Show this help
  quit, q          Exit`
	return header + cmds
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder
	if m.output != "" {
		s.WriteString(m.output)
		s.WriteString("\n\n")
	}

	prompt := promptStyle.Render("jukebox> ")
	s.WriteString(prompt)
	// Render input with cursor
	if m.cursor < len(m.input) {
		s.WriteString(m.input[:m.cursor])
		s.WriteString(lipgloss.NewStyle().Reverse(true).Render(string(m.input[m.cursor])))
		s.WriteString(m.input[m.cursor+1:])
	} else {
		s.WriteString(m.input)
		s.WriteString(lipgloss.NewStyle().Reverse(true).Render(" "))
	}

	return s.String()
}

func main() {
	// Non-interactive mode: jukebox -c "command"
	if len(os.Args) >= 3 && os.Args[1] == "-c" {
		m := initialModel()
		result := m.execute(strings.Join(os.Args[2:], " "))
		fmt.Println(result)
		return
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
