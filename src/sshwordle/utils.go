package sshwordle

import (
	"fmt"
	"regexp"
)

var keys = []string{
	"qwertyuiop",
	"asdfghjkl",
	"zxcvbnm",
}

func makeKeyboard() map[string]guessColor {
	keyboard := make(map[string]guessColor)
	for l := 'a'; l <= 'z'; l++ {
		keyboard[fmt.Sprintf("%c", l)] = white
	}
	return keyboard
}

var DIFFICULTY = 5

func makeGuessesSlice() [][]*Guess {
	guesses := make([][]*Guess, DIFFICULTY+1)
	for i := range guesses {
		guesses[i] = make([]*Guess, DIFFICULTY)
		for a := range guesses[i] {
			guesses[i][a] = &Guess{
				letter: "",
				color:  white,
			}
		}
	}
	return guesses
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

func isLetter(key string) bool {
	match, err := regexp.Match("^[a-z]$", []byte(key))
	if err != nil {
		return false
	}
	return match
}
