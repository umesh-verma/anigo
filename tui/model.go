package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umesh-verma/anigo/player"
	"github.com/umesh-verma/anigo/sources"
	"github.com/umesh-verma/anigo/sources/wpanime"
	"github.com/umesh-verma/anigo/streams" // Fix import path
)

var (
	docStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700"))

	searchBarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(0, 1).
			Width(30)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(0, 1).
			MarginLeft(2)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{
			Light: "#626262",
			Dark:  "#DDD",
		})
)

type ListItem struct {
	title       string
	description string
	url         string
}

func (i ListItem) Title() string       { return i.title }
func (i ListItem) Description() string { return i.description }
func (i ListItem) FilterValue() string { return i.title }

type sourceItem struct {
	name string
	id   string
}

func (i sourceItem) Title() string       { return i.name }
func (i sourceItem) Description() string { return fmt.Sprintf("Source ID: %s", i.id) }
func (i sourceItem) FilterValue() string { return i.name }

type showItem struct {
	title       string
	description string
	url         string
	thumbnail   string
}

func (i showItem) Title() string       { return i.title }
func (i showItem) Description() string { return i.description }
func (i showItem) FilterValue() string { return i.title }

type providerItem struct {
	name      string
	embedURL  string
	processor string
}

func (i providerItem) Title() string       { return i.name }
func (i providerItem) Description() string { return fmt.Sprintf("Provider: %s", i.processor) }
func (i providerItem) FilterValue() string { return i.name }

type qualityItem struct {
	resolution string
	size       int
	url        string
}

func (i qualityItem) Title() string {
	return fmt.Sprintf("%s (%d MB)", i.resolution, i.size/1024/1024)
}
func (i qualityItem) Description() string { return i.url }
func (i qualityItem) FilterValue() string { return i.resolution }

type screen int

const (
	sourceSelect screen = iota
	showList
	episodeList
	providerList
	qualityList
)

type Model struct {
	currentScreen screen
	sources       map[string]sources.SourceProvider
	sourceList    list.Model
	searchInput   textinput.Model
	list          list.Model
	selectedID    string
	selected      any
	err           error
	width         int
	height        int
	showList      list.Model
	shows         []sources.ShowInfo
	episodeList   list.Model
	episodes      []sources.EpisodeInfo
	spinner       spinner.Model
	loading       bool
	loadingMsg    string
	providerList  list.Model
	providers     []sources.StreamProvider
	qualityList   list.Model
	qualities     []streams.VideoData
	player        *player.Player
}

// Change New to return *Model
func New(sources map[string]sources.SourceProvider, defaultPlayer string, playerPaths map[string]string) *Model {
	items := make([]list.Item, 0, len(sources))
	for id := range sources { // simplified range
		items = append(items, sourceItem{
			name: id,
			id:   id,
		})
	}

	sourceList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	sourceList.Title = "Available Sources"
	sourceList.SetShowHelp(true)

	input := textinput.New()
	input.Placeholder = "Type to search anime..." // Update placeholder
	input.Width = 30

	showList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	showList.Title = "Shows"
	showList.SetShowHelp(true)

	episodeList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	episodeList.Title = "Episodes"
	episodeList.SetShowHelp(true)

	providerList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	providerList.Title = "Stream Providers"
	providerList.SetShowHelp(true)

	qualityList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	qualityList.Title = "Available Qualities"
	qualityList.SetShowHelp(true)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

	return &Model{
		currentScreen: sourceSelect,
		sources:       sources,
		sourceList:    sourceList,
		searchInput:   input,
		list:          list.New(nil, list.NewDefaultDelegate(), 0, 0),
		showList:      showList,
		episodeList:   episodeList,
		spinner:       s,
		loading:       false,
		providerList:  providerList,
		qualityList:   qualityList,
		player:        player.New(defaultPlayer, playerPaths),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { // changed to pointer receiver
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd, m.spinner.Tick)
		}
	case error:
		m.err = msg
		m.loading = false
		return m, nil
	case []streams.VideoData: // Add this case to handle video data
		return m.updateQualityList(msg)
	}

	var cmd tea.Cmd
	switch m.currentScreen {
	case sourceSelect:
		return m.updateSourceSelect(msg)
	case showList:
		return m.updateShowList(msg)
	case episodeList:
		return m.updateEpisodeList(msg)
	case providerList:
		return m.updateProviderList(msg)
	case qualityList:
		return m.updateQualityList(msg)
	}

	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSourceSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		searchWidth := 30
		listWidth := m.width - searchWidth - 6 // account for padding and borders
		m.sourceList.SetSize(listWidth, m.height-4)
		return m, nil

	case tea.KeyMsg:
		if m.searchInput.Focused() {
			switch msg.Type {
			case tea.KeyTab, tea.KeyEsc:
				m.searchInput.Blur()
				return m, nil
			case tea.KeyEnter:
				m.loading = true
				m.loadingMsg = "Searching..."
				if i, ok := m.sourceList.SelectedItem().(sourceItem); ok {
					m.selectedID = i.id
					m.currentScreen = showList
					return m, m.performSearch()
				}
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		} else {
			switch msg.String() {
			case "tab": // Changed from "/" to "tab"
				m.searchInput.Focus()
				return m, nil
			case "enter":
				if i, ok := m.sourceList.SelectedItem().(sourceItem); ok {
					m.selectedID = i.id
					if m.searchInput.Value() != "" {
						m.loading = true
						m.loadingMsg = "Searching..."
						m.currentScreen = showList
						return m, m.performSearch()
					}
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.sourceList, cmd = m.sourceList.Update(msg)
	return m, cmd
}

// Remove filterSources function as we don't need it anymore

func (m *Model) performSearch() tea.Cmd { // changed to return tea.Cmd
	return func() tea.Msg {
		if provider, ok := m.sources[m.selectedID]; ok {
			shows, err := provider.Search(m.searchInput.Value())
			if err != nil {
				m.loading = false
				return err
			}
			m.loading = false
			return shows
		}
		m.loading = false
		return fmt.Errorf("no provider found for ID: %s", m.selectedID)
	}
}

func (m *Model) updateShowList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		m.err = msg
		m.currentScreen = sourceSelect
		return m, nil

	case []sources.ShowInfo:
		m.shows = msg
		items := make([]list.Item, len(msg))
		for i, show := range msg {
			// Clean up the title and create a clear description
			title := strings.TrimSpace(show.Title)
			if title == "" {
				title = "Untitled Show"
			}

			items[i] = showItem{
				title:       title,
				description: fmt.Sprintf("Link: %s", strings.TrimPrefix(show.URL, m.sources[m.selectedID].(*wpanime.WPAnimeSource).BaseURL)),
				url:         show.URL,
				thumbnail:   show.Thumbnail,
			}
		}

		// Update the list
		m.showList.Title = fmt.Sprintf("%d Results Found", len(msg))
		m.showList.SetItems(items)

		// Make sure window size is set
		if m.width > 0 {
			m.showList.SetSize(m.width-4, m.height-6)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.showList.SetSize(m.width, m.height-4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.showList.SelectedItem().(showItem); ok {
				m.currentScreen = episodeList
				m.loading = true
				m.loadingMsg = "Loading episodes..."
				return m, m.fetchEpisodes(i.url)
			}
		case "esc":
			m.currentScreen = sourceSelect
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.showList, cmd = m.showList.Update(msg)
	return m, cmd
}

func (m *Model) fetchEpisodes(url string) tea.Cmd {
	return func() tea.Msg {
		episodes, err := m.sources[m.selectedID].GetEpisodes(url)
		m.loading = false
		if err != nil {
			return err
		}
		return episodes
	}
}

func (m *Model) updateEpisodeList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []sources.EpisodeInfo:
		m.episodes = msg
		items := make([]list.Item, len(msg))
		for i, ep := range msg {
			// Clean up and format the episode title/number for display
			epTitle := strings.TrimSpace(ep.Title)
			epNum := strings.TrimSpace(ep.Number)
			epDate := strings.TrimSpace(ep.Date)

			items[i] = ListItem{
				title:       fmt.Sprintf("Episode %s: %s", epNum, epTitle),
				description: fmt.Sprintf("Released: %s", epDate),
				url:         ep.URL,
			}
		}
		m.episodeList.Title = fmt.Sprintf("Episodes (%d)", len(msg))
		m.episodeList.SetItems(items)
		m.loading = false

		// Ensure proper sizing
		if m.width > 0 {
			m.episodeList.SetSize(m.width-4, m.height-6)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.episodeList.SetSize(m.width-4, m.height-6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.episodeList.SelectedItem().(ListItem); ok {
				m.currentScreen = providerList
				return m, func() tea.Msg {
					providers, err := m.sources[m.selectedID].GetStreamProviders(i.url)
					if err != nil {
						return err
					}
					return providers
				}
			}
		case "esc":
			m.currentScreen = showList
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.episodeList, cmd = m.episodeList.Update(msg)
	return m, cmd
}

func (m *Model) updateProviderList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []sources.StreamProvider:
		m.providers = msg
		items := make([]list.Item, len(msg))
		for i, p := range msg {
			items[i] = providerItem{
				name:      p.Name,
				embedURL:  p.EmbedURL,
				processor: p.Processor,
			}
		}
		m.providerList.Title = fmt.Sprintf("Stream Providers (%d)", len(msg))
		m.providerList.SetItems(items)
		m.loading = false

		if m.width > 0 {
			m.providerList.SetSize(m.width-4, m.height-6)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.providerList.SetSize(m.width-4, m.height-6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.providerList.SelectedItem().(providerItem); ok {
				m.currentScreen = qualityList
				m.loading = true
				m.loadingMsg = "Loading qualities..."
				return m, m.processProvider(i)
			}
		case "esc":
			m.currentScreen = episodeList
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.providerList, cmd = m.providerList.Update(msg)
	return m, cmd
}

func (m *Model) processProvider(provider providerItem) tea.Cmd {
	m.loading = true // Set loading state
	return func() tea.Msg {
		switch provider.processor {
		case "rumble":
			videos, err := streams.ProcessRumbleEmbed(provider.embedURL)
			if err != nil {
				return err
			}
			return videos
		// Add other providers as needed...
		default:
			return fmt.Errorf("unsupported provider: %s", provider.processor)
		}
	}
}

func (m *Model) updateQualityList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []streams.VideoData:
		m.qualities = msg
		items := make([]list.Item, len(msg))
		for i, q := range msg {
			items[i] = qualityItem{
				resolution: q.Resolution,
				size:       q.Size,
				url:        q.URL,
			}
		}
		m.qualityList.Title = fmt.Sprintf("Available Qualities (%d)", len(msg))
		m.qualityList.SetItems(items)
		m.loading = false

		// Make sure window size is set
		if m.width > 0 {
			m.qualityList.SetSize(m.width-4, m.height-6)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.qualityList.SelectedItem().(qualityItem); ok {
				// Play the video and quit
				m.selected = &streams.VideoData{
					URL:        i.url,
					Size:       i.size,
					Resolution: i.resolution,
				}
				return m, tea.Sequence(
					func() tea.Msg {
						err := m.player.Play(i.url)
						if err != nil {
							return fmt.Errorf("failed to play video: %v", err)
						}
						return nil
					},
					tea.Quit,
				)
			}
		}
	}

	var cmd tea.Cmd
	m.qualityList, cmd = m.qualityList.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	if m.loading {
		return docStyle.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				titleStyle.Render(m.loadingMsg),
				m.spinner.View(),
			),
		)
	}

	switch m.currentScreen {
	case sourceSelect:
		// Create horizontal layout
		searchSection := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Search Anime"),
			searchBarStyle.Render(m.searchInput.View()),
		)

		listSection := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Select Source"),
			listStyle.Render(m.sourceList.View()),
		)

		// Join sections horizontally
		content := lipgloss.JoinHorizontal(lipgloss.Top,
			searchSection,
			listSection,
		)

		// Help text at bottom
		helpText := "tab: focus search • ↑/↓: navigate • enter: select • q: quit" // Updated help text
		if m.searchInput.Focused() {
			helpText = "enter: search • tab/esc: unfocus • q: quit" // Updated help text
		}

		return docStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				content,
				statusMessageStyle.Render(helpText),
			),
		)
	case showList:
		if len(m.shows) == 0 {
			return docStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					titleStyle.Render("No results found"),
					"\nSearch term: "+m.searchInput.Value(),
					"\nPress 'esc' to go back",
				),
			)
		}

		var s strings.Builder
		s.WriteString(titleStyle.Render(fmt.Sprintf("Search Results for: %s", m.searchInput.Value())))
		s.WriteString("\n\n")

		listView := listStyle.Copy().
			Width(m.width - 4).
			Render(m.showList.View())

		s.WriteString(listView)
		s.WriteString("\n")
		s.WriteString(statusMessageStyle.Render("↑/↓: navigate • enter: select show • esc: back • q: quit"))
		return docStyle.Render(s.String())

	case episodeList:
		if len(m.episodes) == 0 {
			return docStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					titleStyle.Render("No episodes found"),
					"\nPress 'esc' to go back",
				),
			)
		}

		var s strings.Builder
		s.WriteString(titleStyle.Render("Episodes List"))
		s.WriteString("\n\n")

		// Apply full-width styling to the episode list
		listView := listStyle.Copy().
			Width(m.width - 4).
			Render(m.episodeList.View())

		s.WriteString(listView)
		s.WriteString("\n")
		s.WriteString(statusMessageStyle.Render("↑/↓: navigate • enter: watch episode • esc: back • q: quit"))

		return docStyle.Render(s.String())

	case providerList:
		if len(m.providers) == 0 {
			return docStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					titleStyle.Render("No stream providers found"),
					"\nPress 'esc' to go back",
				),
			)
		}

		var s strings.Builder
		s.WriteString(titleStyle.Render("Stream Providers"))
		s.WriteString("\n\n")

		listView := listStyle.Copy().
			Width(m.width - 4).
			Render(m.providerList.View())

		s.WriteString(listView)
		s.WriteString("\n")
		s.WriteString(statusMessageStyle.Render("↑/↓: navigate • enter: select quality • esc: back • q: quit"))

		return docStyle.Render(s.String())

	case qualityList:
		if len(m.qualities) == 0 {
			return docStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					titleStyle.Render("No qualities available"),
					"\nPress 'esc' to go back",
				),
			)
		}

		var s strings.Builder
		s.WriteString(titleStyle.Render("Select Quality"))
		s.WriteString("\n\n")
		s.WriteString(listStyle.Copy().
			Width(m.width - 4).
			Render(m.qualityList.View()))
		s.WriteString("\n")
		s.WriteString(statusMessageStyle.Render("↑/↓: navigate • enter: select • esc: back • q: quit"))
		return docStyle.Render(s.String())

	default:
		return docStyle.Render(titleStyle.Render("Loading..."))
	}
}

// Add this method to expose the selected field
func (m *Model) Selected() *streams.VideoData {
	return m.selected.(*streams.VideoData)
}
