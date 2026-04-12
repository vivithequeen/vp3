package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/qeesung/image2ascii/convert"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var currentStreamer beep.StreamCloser

type model struct {
	table    table.Model
	albumArt string
}

func (m model) Init() tea.Cmd { return nil }

func SwapMusicTo(fp string) {
	f, err := os.Open(fp)
	if err != nil {
		log.Println(err)
		return
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Println(err)
		return
	}

	speaker.Lock()
	if currentStreamer != nil {
		currentStreamer.Close()
	}
	currentStreamer = streamer
	speaker.Unlock()

	resampled := beep.Resample(4, format.SampleRate, beep.SampleRate(44100), streamer)
	speaker.Play(resampled)
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":

			fp := m.table.SelectedRow()[4]
			f, err := os.Open(fp)
			if err != nil {
				break
			}
			defer f.Close()

			tags, err := tag.ReadFrom(f)
			if err != nil {
				break
			}

			pic := tags.Picture()
			if pic == nil {
				break
			}

			img, _, err := image.Decode(bytes.NewReader(pic.Data))
			if err != nil {
				break
			}

			convertOptions := convert.DefaultOptions
			convertOptions.FixedWidth = 25
			convertOptions.FixedHeight = 10

			converter := convert.NewImageConverter()
			s := converter.Image2ASCIIString(img, &convertOptions)

			s += "\n" + tags.Title()
			m.albumArt = s
			go SwapMusicTo(m.table.SelectedRow()[4])
			return m, nil
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	tableView := baseStyle.Render(m.table.View())
	buffer := strings.Repeat("\n     ", 20)
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		tableView, buffer, m.albumArt)
	return tea.NewView(content + "\n  " + m.table.HelpView() +
		"\n")
}

func main() {

	speaker.Init(beep.SampleRate(44100), 4410)

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

			if err != nil {
				log.Fatal(err)
			}

			rows = append(rows, table.Row{"1", m.Title(), m.Artist(), m.Album(), fp})
		}
		return nil
	})

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithWidth(64),
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

	m := model{table: t}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}
