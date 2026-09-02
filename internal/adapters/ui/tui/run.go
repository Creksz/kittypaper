package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"kittypaper/internal/app/dto"
	"kittypaper/internal/app/service"
	"kittypaper/internal/domain/kitty"
	"kittypaper/internal/domain/wallpaper"
)

type wallpaperItem struct {
	item wallpaper.Item
}

func (w wallpaperItem) FilterValue() string { return w.item.Path }
func (w wallpaperItem) Title() string       { return filepath.Base(w.item.Path) }
func (w wallpaperItem) Description() string { return w.item.Path }

type applyMsg struct {
	result dto.ApplyResult
	err    error
}

type refreshMsg struct {
	items []wallpaper.Item
	err   error
}

type model struct {
	list         list.Model
	svc          *service.WallpaperService
	reloadMethod kitty.ReloadMethod
	statusLine   string
	message      string
	width        int
	height       int
	applying     bool
	quitting     bool
}

var (
	applyKey   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply"))
	randomKey  = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "random"))
	refreshKey = key.NewBinding(key.WithKeys("R", "e", "f5"), key.WithHelp("R/e/F5", "refresh"))
)

func newModel(svc *service.WallpaperService, reloadMethod kitty.ReloadMethod, items []wallpaper.Item) model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{applyKey, randomKey}
	}
	delegate.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{{applyKey, randomKey, refreshKey}}
	}

	entries := make([]list.Item, 0, len(items))
	for _, item := range items {
		entries = append(entries, wallpaperItem{item: item})
	}

	l := list.New(entries, delegate, 0, 0)
	l.Title = "Kittypaper"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{refreshKey}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{refreshKey}
	}

	return model{
		list:         l,
		svc:          svc,
		reloadMethod: reloadMethod,
		statusLine:   fmt.Sprintf("%d wallpapers", len(items)),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) applySelected() tea.Cmd {
	selected, ok := m.list.SelectedItem().(wallpaperItem)
	if !ok {
		return nil
	}
	return m.applyPath(selected.item.Path)
}

func (m model) applyPath(path string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.SetWallpaper(context.Background(), dto.SetWallpaperRequest{
			Path:         path,
			ReloadMethod: m.reloadMethod,
		})
		return applyMsg{result: result, err: err}
	}
}

func (m model) applyRandom() tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.SetRandom(context.Background(), m.reloadMethod)
		return applyMsg{result: result, err: err}
	}
}

func (m model) refresh() tea.Cmd {
	return func() tea.Msg {
		items, err := m.svc.ListWallpapers(context.Background())
		return refreshMsg{items: items, err: err}
	}
}

func itemsToListEntries(items []wallpaper.Item) []list.Item {
	entries := make([]list.Item, 0, len(items))
	for _, item := range items {
		entries = append(entries, wallpaperItem{item: item})
	}
	return entries
}

func (m model) formatStatusLine(count int) string {
	if status, err := m.svc.Status(context.Background()); err == nil && status.ActivePath != "" {
		return fmt.Sprintf("%d wallpapers · active: %s", count, status.ActivePath)
	}
	return fmt.Sprintf("%d wallpapers", count)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil
	case applyMsg:
		m.applying = false
		if msg.err != nil {
			m.message = "error: " + msg.err.Error()
			return m, nil
		}
		m.message = "applied " + filepath.Base(msg.result.WallpaperPath)
		if msg.result.Warning != "" {
			m.message += " (warning: " + msg.result.Warning + ")"
		}
		m.statusLine = m.formatStatusLine(len(m.list.Items()))
		return m, nil
	case refreshMsg:
		m.applying = false
		if msg.err != nil {
			m.message = "error: " + msg.err.Error()
			return m, nil
		}
		selectedPath := ""
		if selected, ok := m.list.SelectedItem().(wallpaperItem); ok {
			selectedPath = selected.item.Path
		}
		cmd := m.list.SetItems(itemsToListEntries(msg.items))
		if selectedPath != "" {
			for i, item := range msg.items {
				if item.Path == selectedPath {
					m.list.Select(i)
					break
				}
			}
		}
		m.message = fmt.Sprintf("refreshed — %d wallpapers", len(msg.items))
		m.statusLine = m.formatStatusLine(len(msg.items))
		return m, cmd
	case tea.KeyMsg:
		if m.applying {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.applying = true
			m.message = "applying..."
			return m, m.applySelected()
		case "r":
			m.applying = true
			m.message = "picking random..."
			return m, m.applyRandom()
		case "R", "e", "f5":
			m.applying = true
			m.message = "refreshing..."
			return m, m.refresh()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.list.View())
	if m.message != "" {
		b.WriteString("\n")
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		if strings.HasPrefix(m.message, "error:") {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		}
		b.WriteString(msgStyle.Render(m.message))
	}
	if m.statusLine != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.statusLine))
	}
	return b.String()
}

func Run(svc *service.WallpaperService, reloadMethod kitty.ReloadMethod) error {
	items, err := svc.ListWallpapers(context.Background())
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("no wallpapers found in configured directories")
	}

	m := newModel(svc, reloadMethod, items)
	m.statusLine = m.formatStatusLine(len(items))

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
