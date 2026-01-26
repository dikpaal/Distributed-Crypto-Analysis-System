package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Available trading pairs
var coins = []struct {
	symbol string
	name   string
	short  string
}{
	{"btcusdt", "Bitcoin (BTC)", "BTC"},
	{"ethusdt", "Ethereum (ETH)", "ETH"},
	{"solusdt", "Solana (SOL)", "SOL"},
	{"bnbusdt", "Binance Coin (BNB)", "BNB"},
	{"xrpusdt", "Ripple (XRP)", "XRP"},
	{"dogeusdt", "Dogecoin (DOGE)", "DOGE"},
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("10")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)

	// Dashboard styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 2)

	priceStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	upStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	downStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			MarginBottom(1)
)

// ============================================
// Coin Selection Model
// ============================================

type selectModel struct {
	cursor   int
	selected string
	done     bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(coins)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = coins[m.cursor].symbol
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	s := titleStyle.Render("Select Cryptocurrency to Track") + "\n\n"

	for i, coin := range coins {
		cursor := "  "
		style := itemStyle
		if m.cursor == i {
			cursor = "▸ "
			style = selectedStyle
		}
		s += style.Render(fmt.Sprintf("%s%s", cursor, coin.name)) + "\n"
	}

	s += helpStyle.Render("\n↑/↓: navigate • enter: select • q: quit")
	return s
}

// RunTUI runs the coin selection TUI
func RunTUI() (string, error) {
	m := selectModel{cursor: 0}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(selectModel)
	if result.selected == "" {
		return "", fmt.Errorf("no coin selected")
	}

	return result.selected, nil
}

// ============================================
// Dashboard Model
// ============================================

// PriceData holds current price information
type PriceData struct {
	Price         float64
	PrevPrice     float64
	High          float64
	Low           float64
	MovingAverage float64
	Change        float64
	ChangePercent float64
	UpdatedAt     time.Time
}

// tickMsg triggers periodic updates
type tickMsg time.Time

// priceMsg carries new price data
type priceMsg PriceData

// Dashboard model
type dashboardModel struct {
	symbol    string
	coinName  string
	coinShort string
	data      PriceData
	history   []float64
	server    *Server
	quitting  bool
}

func newDashboardModel(symbol string, server *Server) dashboardModel {
	name := GetCoinName(symbol)
	short := GetCoinShort(symbol)
	return dashboardModel{
		symbol:    symbol,
		coinName:  name,
		coinShort: short,
		server:    server,
		history:   make([]float64, 0, 20),
	}
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.SetWindowTitle("Trading Pipeline - "+m.coinName))
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m dashboardModel) fetchPrice() tea.Cmd {
	return func() tea.Msg {
		price := m.server.GetPrice()
		ma := float64(GetMovingAverage())
		high := float64(GetHigh())
		low := float64(GetLow())

		var change, changePercent float64
		if m.data.Price > 0 {
			change = price - m.data.Price
			changePercent = (change / m.data.Price) * 100
		}

		return priceMsg{
			Price:         price,
			PrevPrice:     m.data.Price,
			High:          high,
			Low:           low,
			MovingAverage: ma,
			Change:        change,
			ChangePercent: changePercent,
			UpdatedAt:     time.Now(),
		}
	}
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		return m, tea.Batch(m.fetchPrice(), tickCmd())

	case priceMsg:
		m.data = PriceData(msg)
		// Add to history for sparkline
		if msg.Price > 0 {
			m.history = append(m.history, msg.Price)
			if len(m.history) > 20 {
				m.history = m.history[1:]
			}
		}
		return m, nil
	}

	return m, nil
}

func (m dashboardModel) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	// Header
	header := headerStyle.Render(fmt.Sprintf("◆ %s Real-Time Dashboard", m.coinName))

	// Price display
	priceStr := fmt.Sprintf("$%.2f", m.data.Price)
	if m.data.Price >= 1000 {
		priceStr = fmt.Sprintf("$%.2f", m.data.Price)
	} else if m.data.Price < 1 {
		priceStr = fmt.Sprintf("$%.6f", m.data.Price)
	}

	// Change indicator
	var changeStr string
	if m.data.Change > 0 {
		changeStr = upStyle.Render(fmt.Sprintf("▲ +%.2f (+%.2f%%)", m.data.Change, m.data.ChangePercent))
	} else if m.data.Change < 0 {
		changeStr = downStyle.Render(fmt.Sprintf("▼ %.2f (%.2f%%)", m.data.Change, m.data.ChangePercent))
	} else {
		changeStr = labelStyle.Render("━ 0.00 (0.00%)")
	}

	// Main price box
	priceDisplay := priceStyle.Render(priceStr) + "  " + changeStr

	// Stats
	stats := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("Moving Avg:"),
		valueStyle.Render(fmt.Sprintf("$%.2f", m.data.MovingAverage)),
		labelStyle.Render("Session High:"),
		upStyle.Render(fmt.Sprintf("$%.2f", m.data.High)),
		labelStyle.Render("Session Low:"),
		downStyle.Render(fmt.Sprintf("$%.2f", m.data.Low)),
		labelStyle.Render("Spread:"),
		valueStyle.Render(fmt.Sprintf("$%.2f", m.data.High-m.data.Low)),
	)

	// Sparkline (simple ASCII)
	sparkline := m.renderSparkline()

	// Combine
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s%s\n\n%s",
		header,
		priceDisplay,
		stats,
		labelStyle.Render("Price History: "),
		sparkline,
		helpStyle.Render("Press 'q' to quit"),
	)

	return boxStyle.Render(content)
}

func (m dashboardModel) renderSparkline() string {
	if len(m.history) < 2 {
		return labelStyle.Render("waiting for data...")
	}

	// Find min/max
	min, max := m.history[0], m.history[0]
	for _, v := range m.history {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Sparkline chars
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var spark string
	rang := max - min
	if rang == 0 {
		rang = 1
	}

	for i, v := range m.history {
		normalized := (v - min) / rang
		idx := int(normalized * float64(len(chars)-1))
		if idx >= len(chars) {
			idx = len(chars) - 1
		}

		// Color based on trend
		char := string(chars[idx])
		if i > 0 && v > m.history[i-1] {
			spark += upStyle.Render(char)
		} else if i > 0 && v < m.history[i-1] {
			spark += downStyle.Render(char)
		} else {
			spark += valueStyle.Render(char)
		}
	}

	return spark
}

// RunDashboard starts the real-time dashboard
func RunDashboard(symbol string, server *Server) error {
	m := newDashboardModel(symbol, server)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ============================================
// Helper Functions
// ============================================

func GetCoinName(symbol string) string {
	for _, coin := range coins {
		if coin.symbol == symbol {
			return coin.name
		}
	}
	return symbol
}

func GetCoinShort(symbol string) string {
	for _, coin := range coins {
		if coin.symbol == symbol {
			return coin.short
		}
	}
	return symbol
}
