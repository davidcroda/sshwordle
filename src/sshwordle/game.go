package sshwordle

import (
	"database/sql"
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/gliderlabs/ssh"
	"log"
	_ "modernc.org/sqlite"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	baseStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#fefefe"))
	greenStyle  = baseStyle.Copy().Background(lipgloss.Color("#04b575"))
	yellowStyle = baseStyle.Copy().Background(lipgloss.Color("#bd8024"))
	greyStyle   = baseStyle.Copy().Background(lipgloss.Color("#636664"))
	redStyle    = baseStyle.Copy().Background(lipgloss.Color("#d63131"))
	layoutStyle = lipgloss.NewStyle().Align(lipgloss.Center)

	winnerStyle = baseStyle.Copy().Background(lipgloss.Color("#58565e")).
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
	white:  baseStyle,
	red:    redStyle,
}

type Guess struct {
	letter string
	color  guessColor
}

type Game struct {
	backend      Backend
	db           *sql.DB
	complete     bool
	showWord     bool
	currentGuess int
	currentCol   int
	maxGuesses   int
	guesses      [][]*Guess
	identifier   string
	keyboard     map[string]guessColor
	session      ssh.Session
	stopwatch    stopwatch.Model
	viewport     viewport.Model
	height       int
	width        int
	won          bool
	word         string
	results      []GameResult
}

func NewGame(width int, height int, session ssh.Session, backend Backend) Game {
	guesses := makeGuessesSlice()
	db := openDb()
	identifier := makeIdentifier(session)
	keyboard := makeKeyboard()

	return Game{
		backend:    backend,
		db:         db,
		guesses:    guesses,
		height:     height,
		identifier: identifier,
		keyboard:   keyboard,
		maxGuesses: len(guesses),
		stopwatch:  stopwatch.NewWithInterval(time.Second),
		session:    session,
		viewport:   viewport.New(width, height),
		width:      width,
		word:       backend.GetRandomWord(DIFFICULTY),
	}
}

func (g Game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.resizeViewport(msg)
	case tea.KeyMsg:
		return g.handleKeyPress(msg)
	case gameResultsMsg:
		g.results = msg
	case gameCompleteMsg:
		result := GameResult{
			Seconds:    g.stopwatch.Elapsed(),
			GuessCount: g.currentGuess + 1,
			Word:       g.word,
			Timestamp:  time.Now().Second(),
		}
		g.results = append(g.results, result)
		cmds = append(cmds, g.stopwatch.Stop())
		cmds = append(cmds, saveGameResult(g.db, g.identifier, &result))
	case showPostGameMsg:
		g.complete = true
	case invalidMsg:
		g.setAllGuesses(red)
	}

	g.viewport, cmd = g.viewport.Update(msg)
	cmds = append(cmds, cmd)
	g.stopwatch, cmd = g.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	return g, tea.Batch(cmds...)
}

func (g Game) View() string {
	if !g.complete {
		g.viewport.SetContent(g.renderGameBoard())
	} else {
		g.viewport.SetContent(g.renderPostGame())
	}
	return fmt.Sprintf("%s\n%s\n%s", g.headerView(), g.viewport.View(), g.footerView())
}

type gameCompleteMsg struct{}
type invalidMsg struct{}
type showPostGameMsg struct{}

func GameComplete() tea.Msg {
	return gameCompleteMsg{}
}

func InvalidWord() tea.Msg {
	return invalidMsg{}
}

func (g Game) Init() tea.Cmd {
	return tea.Batch(g.stopwatch.Init(), getGameResults(g.db, g.identifier))
}

func (g *Game) setAllGuesses(color guessColor) {
	for i := range g.guesses[g.currentGuess] {
		g.guesses[g.currentGuess][i].color = color
	}
}

func (g Game) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := strings.ToLower(msg.String())
	if k == "ctrl+c" {
		err := g.db.Close()
		if err != nil {
			log.Fatal(err)
		}
		return g, tea.Quit
	} else if k == " " {
		if g.complete {
			ng := NewGame(g.width, g.height, g.session, g.backend)
			return ng, ng.Init()
		}
		return g, nil
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
	return g, nil
}

func (g *Game) handleEnter() tea.Cmd {
	if g.currentCol != len(g.word) || g.currentGuess >= len(g.guesses) {
		return nil
	}

	if !g.backend.ValidateWord(g.guesses[g.currentGuess]) {
		return InvalidWord
	}
	if won := g.gradeCurrentGuess(); won {
		g.won = true
		return GameComplete
	}
	if g.currentGuess >= len(g.guesses)-1 {
		return GameComplete
	}
	g.currentGuess++
	g.currentCol = 0
	return nil
}

func (g *Game) handleBackspace() {
	if g.currentCol > 0 {
		g.currentCol--
		g.guesses[g.currentGuess][g.currentCol].letter = ""
		g.guesses[g.currentGuess][g.currentCol].color = white
	}
}

func (g *Game) handleLetterPress(k string) {
	if g.currentCol < len(g.word) && g.currentGuess < len(g.guesses) {
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

	for i := range guess {
		if guess[i].letter == string(word[i]) {
			g.setGuessColor(guess[i], green)
			used[i] = true
		}
	}
	if len(used) == len(word) {
		return true
	}

	for i := range guess {
		if guess[i].color == green {
			continue
		}
		if matched, index := g.checkLetterIncorrectPosition(guess[i], used); matched {
			used[index] = true
		}
	}

	return false
}

func (g *Game) checkLetterIncorrectPosition(guess *Guess, used map[int]bool) (bool, int) {
	for index := range g.word {
		if string(g.word[index]) == guess.letter && !used[index] {
			g.setGuessColor(guess, yellow)
			return true, index
		}
	}
	g.setGuessColor(guess, grey)
	return false, 0
}

func (g Game) headerView() string {
	title := titleStyle.Render("SSH Wordle")
	line := strings.Repeat("─", max(0, g.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (g Game) footerView() string {
	help := helpStyle.Render(fmt.Sprintf("ctrl+c - quit"))
	info := infoStyle.Render(fmt.Sprintf("Guess %d/%d, Seconds: %s", g.currentGuess+1, len(g.guesses), g.stopwatch.View()))
	line := strings.Repeat("─", max(0, g.viewport.Width-lipgloss.Width(info)-lipgloss.Width(help)))
	return lipgloss.JoinHorizontal(lipgloss.Center, help, line, info) + "\n"
}

func (g *Game) resizeViewport(msg tea.WindowSizeMsg) {
	headerHeight := lipgloss.Height(g.headerView())
	footerHeight := lipgloss.Height(g.footerView())
	verticalMarginHeight := headerHeight + footerHeight

	g.viewport.Height = msg.Height - verticalMarginHeight
	g.viewport.Width = msg.Width
	g.viewport.YPosition = headerHeight

	g.height = msg.Height - verticalMarginHeight
	g.width = msg.Width
}

func (g Game) renderPostGame() string {
	output := ""

	if g.won {
		plural := ""
		if g.currentGuess > 0 {
			plural = "es"
		}
		output = fmt.Sprintf(
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
	} else {
		output = fmt.Sprintf("Unlucky. The word was \"%s\". Better luck next time!", g.word)
	}

	output += "\n\nPress [SPACE] to play again..."
	output = g.center(winnerStyle.Render(output))

	return output
}

func (g Game) renderGameBoard() string {

	output := g.renderGrid()

	output += renderKeyboard(g.keyboard)

	if g.showWord {
		output += "\n\nWORD: " + g.word + "\n"
	}

	return g.center(output)
}

func (g Game) renderGrid() string {
	output := " "
	padding := 3
	for row := range g.guesses {
		for col := range g.guesses[row] {
			guess := g.guesses[row][col]
			if guess.letter == "" {
				output += baseStyle.Render(strings.Repeat(" ", padding))
			} else {
				output += styles[guess.color].Render(" " + guess.letter + " ")
			}
			output += strings.Repeat(" ", padding)
		}
		output += "\n\n  "
	}
	return output
}

func (g Game) center(output string) string {
	paddingTop := g.height/2 - lipgloss.Height(output)/2
	return layoutStyle.Width(g.width).PaddingTop(paddingTop).Render(output)
}
