package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	trackStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	albumStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	playStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	pauseStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
)

func FormatTime(ms int) string {
	total := ms / 1000
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func FormatStatus(s *Status) string {
	if s.Stopped || s.Track == nil {
		return "♫ Ready (nothing playing)"
	}

	t := s.Track
	artist := strings.Join(t.ArtistNames, ", ")
	title := trackStyle.Render(fmt.Sprintf("♫ %s — %s", artist, t.Name))
	album := albumStyle.Render("  " + t.AlbumName)

	pos := FormatTime(t.Position)
	dur := FormatTime(t.Duration)

	var state string
	if s.Paused {
		state = pauseStyle.Render("⏸ paused")
	} else {
		state = playStyle.Render("▶ playing")
	}

	info := fmt.Sprintf("  %s / %s  %s", pos, dur, state)

	return title + "\n" + album + "\n" + info
}
