package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

var (
	pink          = lipgloss.Color("#F4A4BF")
	white         = lipgloss.Color("#FFFFFF")
	helpStyle     = lipgloss.NewStyle().Foreground(gray).Render
	iconStyle     = lipgloss.NewStyle().Foreground(pink).Render
	colorGradiant = lipgloss.Blend2D(visBars, visHeight, 45.0, pink, white)
	yellow        = lipgloss.Color("#F4A4BF")

	gray     = lipgloss.Color("#828282")
	helpText = "↑/↓ nagigate | ←/→ seek    | enter play\na/d /      | space pause | n "
)

func applyGradient(s string) string {
	lines := strings.Split(s, "\n")
	var out strings.Builder
	for y, line := range lines {
		if y > 0 {
			out.WriteByte('\n')
		}
		if y >= visHeight {
			out.WriteString(line)
			continue
		}
		for x, r := range []rune(line) {
			if x >= visBars {
				out.WriteRune(r)
				continue
			}
			c := colorGradiant[y*visBars+x]
			out.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(r)))
		}
	}
	return out.String()
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#F4A4BF"))
var coverTheme = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#F4A4BF"))

var tickSpeed time.Duration = 60
var currentStreamer beep.StreamSeekCloser
var currentCtrl *beep.Ctrl
var isPasued bool = false
var currentSampleRate beep.SampleRate
var currentVolume float64 = -7.5 // -7.5 to 2.5 seem to be best
var currentVolumeCtrl *effects.Volume

type tickMsg struct{}
type model struct {
	table            table.Model
	albumArt         string
	musicTitle       string
	currentSongIndex int

	percent        float64
	progress       progress.Model
	songLength     time.Duration
	songElapseTime time.Duration
}

func (m model) Init() tea.Cmd { return tickCmd() }

func createTable() table.Model {
	columns := []table.Column{
		{Title: "#", Width: 3},
		{Title: "Title", Width: 18},
		{Title: "Artist", Width: 18},
		{Title: "Albumn", Width: 18},
		{Title: "Filepath", Width: 0},
	}
	var rows []table.Row
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
	}

	musicDir := filepath.Join(homeDir, "Music")
	index := 0
	filepath.Walk(musicDir, func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		matched, err := filepath.Match("*.mp3", fi.Name())
		if err != nil {
			return err
		}
		if matched {
			f, err := os.Open(fp)
			if err != nil {
				log.Fatal(err)
			}

			m, err := tag.ReadFrom(f)
			f.Close()

			if err != nil {
				return nil
			}
			index++
			rows = append(rows, table.Row{strconv.Itoa(index), m.Title(), m.Artist(), m.Album(), fp})
		}
		return nil
	})

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(19),
		table.WithHeight(19),
		table.WithWidth(66),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	return t
}

func toggleMusicPause() {
	speaker.Lock()
	if currentCtrl != nil {
		currentCtrl.Paused = !currentCtrl.Paused
	}
	speaker.Unlock()
}
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/tickSpeed, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tickMsg:
		if m.songLength != 0 && !isPasued {
			m.percent += float64(time.Second/tickSpeed) / float64(m.songLength)

			m.songElapseTime += time.Second / tickSpeed
		}
		if m.percent >= 1.0 {
			(&m).nextSong()
		}
		return m, tickCmd()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "left":
			m.songElapseTime -= 10 * time.Second
			if m.songElapseTime < 0 {
				m.songElapseTime = 0
			}
			m.percent = float64(m.songElapseTime) / float64(m.songLength)
			seekTo(m.songElapseTime)
			return m, nil
		case "right":
			m.songElapseTime += 10 * time.Second
			if m.songElapseTime > m.songLength {
				m.songElapseTime = m.songLength
			}
			m.percent = float64(m.songElapseTime) / float64(m.songLength)
			seekTo(m.songElapseTime)
			return m, nil

		case "a":
			newVolume := float32(currentVolume) - 0.5
			if newVolume < -7.5 {
				newVolume = -7.5
			}
			setVolume(newVolume)
			return m, nil
			return m, nil
		case "d":
			newVolume := float32(currentVolume) + 0.5
			if newVolume > 2.5 {
				newVolume = 2.5
			}
			setVolume(newVolume)
			return m, nil
			return m, nil
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "space":
			isPasued = !isPasued
			toggleMusicPause()
			return m, nil

		case "enter":
			fp := m.table.SelectedRow()[4]
			idx, _ := strconv.Atoi(m.table.SelectedRow()[0])
			m.currentSongIndex = idx - 1
			m.albumArt, m.musicTitle, m.songLength = changeCurrentSong(fp)
			m.percent = 0
			m.songElapseTime = 0
			return m, nil
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	tableView := m.table.View()
	buffer := strings.Repeat("\n     ", 20)

	fmtDur := func(d time.Duration) string {
		d = d.Round(time.Second)
		return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
	}
	var progress = fmtDur(m.songElapseTime) + "/" + fmtDur(m.songLength)

	b := "\n" + lipgloss.JoinHorizontal(lipgloss.Top, helpText, "                     "+fmt.Sprintf("%.1f", ((currentVolume+7.5)/10)*100)+"%")
	leftTable := tableView + "\n" + helpStyle(b)
	leftTable = baseStyle.Render(leftTable)
	var musicRight = ""

	if m.albumArt != "" {
		artBuffer := strings.Repeat("\n      ", 17)
		a := lipgloss.JoinHorizontal(lipgloss.Top, artBuffer, coverTheme.Render(m.albumArt), artBuffer)
		// 6 (buffer) + 1 (inner border) + 37 (art) + 1 (inner border) + 6 (buffer) = 51
		centerStyle := lipgloss.NewStyle().Width(51).AlignHorizontal(lipgloss.Center)
		titleText := strings.TrimLeft(m.musicTitle, "\n")
		icon := " "
		if currentCtrl != nil && currentCtrl.Paused {
			icon = "󰏤 "
		}

		musicRight = a + "\n\n" + centerStyle.Render(titleText) + "\n" + centerStyle.Render(iconStyle(icon)+m.progress.ViewAs(m.percent)) + "\n" + centerStyle.Render(progress) + "\n"
		musicRight = coverTheme.Render(musicRight)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		leftTable, buffer, musicRight)
	return tea.NewView(content + "\n" + applyGradient(getVisualizer()) + "\n")
}

func main() {

	speaker.Init(beep.SampleRate(44100), 4410)

	t := createTable()

	prog := progress.New(progress.WithWidth(37), progress.WithScaled(true), progress.WithColors(pink, yellow), progress.WithoutPercentage())
	m := model{table: t, progress: prog}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}
