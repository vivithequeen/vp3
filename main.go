package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dhowden/tag"
	"github.com/qeesung/image2ascii/convert"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

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
			convertOptions.FixedWidth = 50
			convertOptions.FixedHeight = 20

			converter := convert.NewImageConverter()
			return m, tea.Println(converter.Image2ASCIIString(img, &convertOptions))
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n")
}

func main() {

	// f,err := os.Open("output.mp3")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// streamer, format, err := mp3.Decode(f)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer streamer.Close()

	// speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// speaker.Play(streamer)

	// select {}
	columns := []table.Column{
		{Title: "#", Width: 3},
		{Title: "Title", Width: 18},
		{Title: "Artist", Width: 18},
		{Title: "Albumn", Width: 18},
		{Title: "Filepath", Width: 0},
	}
	var rows []table.Row
	filepath.Walk("/home/violet/Music", func(fp string, fi os.FileInfo, err error) error {
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

	m := model{t}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}
