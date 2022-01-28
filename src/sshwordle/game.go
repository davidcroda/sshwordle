package sshwordle

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/gliderlabs/ssh"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/kv"
	"github.com/charmbracelet/lipgloss"
)

type guessColor int

const (
	green guessColor = iota
	yellow
	grey
	white
	red
)

var (
	style = lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#fefefe"))
	greenStyle  = style.Copy().Background(lipgloss.Color("#04b575"))
	yellowStyle = style.Copy().Background(lipgloss.Color("#bd8024"))
	greyStyle   = style.Copy().Background(lipgloss.Color("#636664"))
	redStyle    = style.Copy().Background(lipgloss.Color("#d63131"))
	layoutStyle = lipgloss.NewStyle().Align(lipgloss.Center)

	winnerStyle = style.Copy().Background(lipgloss.Color("#58565e")).
			Foreground(lipgloss.Color("#fefefe")).
			BorderStyle(lipgloss.DoubleBorder()).
			Padding(5).
			Bold(true)

	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()
	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
	helpStyle = titleStyle.Copy().Foreground(lipgloss.AdaptiveColor{
		Light: "#909090",
		Dark:  "#626262",
	})
)

var styles = map[guessColor]lipgloss.Style{
	green:  greenStyle,
	yellow: yellowStyle,
	grey:   greyStyle,
	white:  style,
	red:    redStyle,
}

type Guess struct {
	letter string
	color  guessColor
}

type Game struct {
	backend      Backend
	currentGuess int
	currentCol   int
	guesses      [][]*Guess
	height       int
	keyboard     map[string]guessColor
	maxGuesses   int
	session      ssh.Session
	showWord     bool
	stopwatch    stopwatch.Model
	viewport     viewport.Model
	width        int
	won          bool
	word         string
	results      []GameResult
}

func makeKeyboard() map[string]guessColor {
	keyboard := make(map[string]guessColor)
	for l := 'a'; l <= 'z'; l++ {
		keyboard[fmt.Sprintf("%c", l)] = white
	}
	return keyboard
}

var difficulty = 5

func NewGame(width int, height int, session ssh.Session, backend Backend) Game {
	guesses := make([][]*Guess, difficulty+1)
	for i := range guesses {
		guesses[i] = make([]*Guess, difficulty)
		for a := range guesses[i] {
			guesses[i][a] = &Guess{
				letter: "",
				color:  white,
			}
		}
	}

	return Game{
		backend:    backend,
		guesses:    guesses,
		height:     height,
		keyboard:   makeKeyboard(),
		maxGuesses: len(guesses),
		stopwatch:  stopwatch.NewWithInterval(time.Second),
		session:    session,
		viewport:   viewport.New(width, height),
		width:      width,
		word:       backend.GetRandomWord(difficulty),
	}
}

type wonMsg struct{}
type invalidMsg struct{}
type showPostGameMsg struct{}

func Won() tea.Msg {
	return wonMsg{}
}

func InvalidWord() tea.Msg {
	return invalidMsg{}
}

func (g Game) Identifier() string {
	network := strings.Split(g.session.RemoteAddr().String(), ":")
	return network[0]
}

func (g Game) Init() tea.Cmd {
	return tea.Batch(g.stopwatch.Init(), getGameResults(g.Identifier()))
}

func isLetter(key string) bool {
	match, err := regexp.Match("^[a-z]$", []byte(key))
	if err != nil {
		return false
	}
	return match
}

func (g *Game) setAllGuesses(color guessColor) {
	for i := range g.guesses[g.currentGuess] {
		g.guesses[g.currentGuess][i].color = color
	}
}

func (g *Game) handleEnter() tea.Cmd {
	if g.currentCol != len(g.word) || g.currentGuess >= g.maxGuesses-1 {
		return nil
	}
	current := g.guesses[g.currentGuess][g.currentCol-1]
	if current == nil || current.letter == "" {
		return nil
	}
	if !g.backend.ValidateWord(g.guesses[g.currentGuess]) {
		return InvalidWord
	}
	if won := g.gradeCurrentGuess(); won {
		return Won
	}
	g.currentGuess++
	g.currentCol = 0
	return nil
}

func (g *Game) handleBackspace() {
	if g.currentCol > 0 {
		g.currentCol--
		g.guesses[g.currentGuess][g.currentCol].letter = ""
	} else {
		g.setAllGuesses(white)
	}
}

func (g *Game) handleLetterPress(k string) {
	if g.currentCol < len(g.word) {
		g.guesses[g.currentGuess][g.currentCol].letter = k
		g.currentCol++
	}
}

func (g *Game) setGuessColor(guess *Guess, color guessColor) {
	guess.color = color
	if color < g.keyboard[guess.letter] {
		g.keyboard[guess.letter] = color
	}
}

func (g *Game) gradeCurrentGuess() bool {
	word := g.word
	guess := g.guesses[g.currentGuess]
	used := make(map[int]bool)
	matched := 0

	for i := range guess {
		if guess[i].letter == string(word[i]) {
			g.setGuessColor(guess[i], green)
			used[i] = true
			matched++
		}
	}
	if matched == len(word) {
		return true
	}

nextGuess:
	for i := range guess {
		if guess[i].color == green {
			continue
		}
		letter := guess[i].letter
		for a := range word {
			if string(word[a]) == letter && !used[a] {
				g.setGuessColor(guess[i], yellow)
				used[a] = true
				continue nextGuess
			}
		}
		g.setGuessColor(guess[i], grey)
	}

	return false
}

var keys = []string{
	"qwertyuiop",
	"asdfghjkl",
	"zxcvbnm",
}

func renderKeyboard(keyboard map[string]guessColor) string {
	output := "\n\n"
	for i := range keys {
		for a := range keys[i] {
			letter := string(keys[i][a])
			color := keyboard[letter]
			output += styles[color].
				Render(" " + letter + " ")
		}
		output += "\n"
	}
	return output + "\n\n"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g Game) headerView() string {
	title := titleStyle.Render("SSH Wordle")
	line := strings.Repeat("─", max(0, g.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (g Game) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("Guess %d/%d, Elapsed: %s", g.currentGuess+1, g.maxGuesses, g.stopwatch.View()))
	help := helpStyle.Render(fmt.Sprintf("ctrl+c - quit"))
	line := strings.Repeat("─", max(0, g.viewport.Width-lipgloss.Width(info)-lipgloss.Width(help)))
	return lipgloss.JoinHorizontal(lipgloss.Center, help, line, info)
}

func (g Game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(g.headerView())
		footerHeight := lipgloss.Height(g.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		g.viewport.Height = msg.Height - verticalMarginHeight
		g.viewport.Width = msg.Width
		g.viewport.YPosition = headerHeight

		g.height = msg.Height - verticalMarginHeight
		g.width = msg.Width
	case tea.KeyMsg:
		k := strings.ToLower(msg.String())
		if k == "ctrl+c" {
			return g, tea.Quit
		} else if g.won {
			return g, nil
		} else if k == "*" {
			g.showWord = !g.showWord
			return g, nil
		} else if k == "backspace" {
			g.handleBackspace()
			return g, nil
		} else if k == "enter" {
			cmd := g.handleEnter()
			return g, cmd
		} else if isLetter(k) {
			g.handleLetterPress(k)
			return g, nil
		}
	case gameResultsMsg:
		g.results = msg
	case wonMsg:
		result := GameResult{
			Elapsed:    g.stopwatch.Elapsed(),
			GuessCount: g.currentGuess + 1,
			Word:       g.word,
			Date:       time.Now(),
		}
		g.results = append(g.results, result)
		cmds = append(cmds, g.stopwatch.Stop())
		cmds = append(cmds, g.saveGameResult())
	case showPostGameMsg:
		g.won = true
	case invalidMsg:
		g.setAllGuesses(red)
	}

	g.viewport, cmd = g.viewport.Update(msg)
	cmds = append(cmds, cmd)
	g.stopwatch, cmd = g.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	return g, tea.Batch(cmds...)
}

func (g Game) saveGameResult() tea.Cmd {
	return func() tea.Msg {
		g.saveGameResults()
		return showPostGameMsg{}
	}
}

func (g Game) View() string {
	if g.results == nil {
		g.viewport.SetContent(fmt.Sprintf("Welcome %s, Loading your stats...", g.Identifier()))
	} else if !g.won {
		g.viewport.SetContent(g.renderGameBoard())
	} else {
		g.viewport.SetContent(g.renderPostGame())
	}
	return fmt.Sprintf("%s\n%s\n%s", g.headerView(), g.viewport.View(), g.footerView())
}

type GameResult struct {
	Elapsed    time.Duration `json:"elapsed"`
	GuessCount int           `json:"guessCount"`
	Word       string        `json:"word"`
	Date       time.Time     `json:"date"`
}

const dbName = "game-results-db"

func (g *Game) saveGameResults() {
	resultsBytes, err := json.Marshal(g.results)
	if err != nil {
		log.Fatalln(err)
	}

	db, err := kv.OpenWithDefaults(dbName)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.Set([]byte(g.Identifier()), resultsBytes)
	if err != nil {
		log.Fatalln(err)
	}
}

type gameResultsMsg []GameResult

func getGameResults(id string) tea.Cmd {
	return func() tea.Msg {
		db, err := kv.OpenWithDefaults(dbName)
		if err != nil {
			log.Fatalln(err)
		}
		defer db.Close()

		if err := db.Sync(); err != nil {
			log.Fatalln(err)
		}

		value, err := db.Get([]byte(id))
		if err != nil && err.Error() == "Key not found" {
			value = []byte("[]")
		} else if err != nil {
			log.Fatalln(err)
		}

		log.Printf("%s\n", value)

		var results []GameResult
		err = json.Unmarshal(value, &results)
		if err != nil {
			log.Fatalln(err)
		}

		return gameResultsMsg(results)
	}
}

type GameStats struct {
	GuessCounts  []int
	TotalGuesses float64
	TotalTime    float64
	Count        float64
	AverageGuess float64
	AverageTime  float64
}

func (g Game) getGameStats() *GameStats {
	guessCount := make([]int, 6)
	totalGuesses := 0.0
	totalTime := 0.0
	count := 0.0
	for i := range g.results {
		result := g.results[i]
		guessCount[result.GuessCount-1]++
		totalGuesses += float64(result.GuessCount)
		totalTime += result.Elapsed.Seconds()
		count++
	}
	averageGuess := totalGuesses / count
	averageTime := totalTime / count
	return &GameStats{
		GuessCounts:  guessCount,
		TotalGuesses: totalGuesses,
		TotalTime:    totalTime,
		Count:        count,
		AverageGuess: averageGuess,
		AverageTime:  averageTime,
	}
}

func (g Game) renderPostGame() string {
	plural := ""
	if g.currentGuess > 0 {
		plural = "es"
	}

	output := fmt.Sprintf(
		"Congrats! You won in %s with %d guess%s!\n\n",
		g.stopwatch.Elapsed().String(),
		g.currentGuess+1,
		plural,
	)

	stats := g.getGameStats()
	output += fmt.Sprintf("Average Guess: %.2f\n", stats.AverageGuess)
	output += fmt.Sprintf("Average Seconds: %.2f\n", stats.AverageTime)
	output += "\n\n"

	for i := range stats.GuessCounts {
		prog := progress.New(progress.WithDefaultScaledGradient())
		prog.Width = g.width / 4
		output += fmt.Sprintf("%d: %s\n", i+1, prog.ViewAs(float64(stats.GuessCounts[i])/stats.Count))
	}

	output = g.center(winnerStyle.Render(output))

	return output
}

func (g Game) renderGameBoard() string {
	output := "  "

	padding := 3
	for row := range g.guesses {
		for col := range g.guesses[row] {
			guess := g.guesses[row][col]
			if guess.letter == "" {
				output += style.Render(strings.Repeat(" ", padding))
			} else {
				output += styles[guess.color].Render(" " + guess.letter + " ")
			}
			output += strings.Repeat(" ", padding)
		}
		output += "\n\n  "
	}

	output += renderKeyboard(g.keyboard)

	if g.showWord {
		output += "\n\nWORD: " + g.word + "\n"
	}

	return g.center(output)
}

func (g Game) center(output string) string {
	paddingTop := g.height/2 - lipgloss.Height(output)/2
	return layoutStyle.Width(g.width).PaddingTop(paddingTop).Render(output)
}
