package sshwordle

import (
	"database/sql"
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"time"
)

func openDb() *sql.DB {
	db, err := sql.Open("sqlite", "file:///app/db.sqlite")
	if err != nil {
		log.Panic(err)
	}
	return db
}

func saveGameResult(db *sql.DB, identifier string, result *GameResult) tea.Cmd {
	return func() tea.Msg {
		saveGameResults(db, identifier, result)
		return showPostGameMsg{}
	}
}

type GameResult struct {
	Seconds    time.Duration `json:"elapsed"`
	GuessCount int           `json:"guessCount"`
	Word       string        `json:"word"`
	Timestamp  int           `json:"date"`
}

func saveGameResults(db *sql.DB, identifier string, result *GameResult) {
	stmt, err := db.Prepare("INSERT INTO games(user_identifier, seconds, guess_count, word, timestamp) VALUES(?,?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(identifier, result.Seconds, result.GuessCount, result.Word, result.Timestamp)
	if err != nil {
		log.Fatal(err)
	}
}

type gameResultsMsg []GameResult

func getGameResults(db *sql.DB, identifier string) tea.Cmd {
	return func() tea.Msg {
		var results []GameResult
		rows, err := db.Query("SELECT seconds, guess_count, word, timestamp FROM games WHERE user_identifier = ?", identifier)
		if err != nil {
			log.Fatal(err)
		}

		defer rows.Close()
		for rows.Next() {
			result := GameResult{}
			err := rows.Scan(&result.Seconds, &result.GuessCount, &result.Word, &result.Timestamp)
			if err != nil {
				log.Fatal(err)
			}
			results = append(results, result)
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
		totalTime += result.Seconds.Seconds()
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
