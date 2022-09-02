package sshwordle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

type Backend interface {
	GetRandomWord(length int) string
	ValidateWord(guess []*Guess) bool
}

type StaticBackend struct {
	words   []string
	allowed map[string]bool
}

func (b StaticBackend) GetRandomWord(_ int) string {
	return b.words[rand.Intn(len(b.words))]
}

func (b StaticBackend) ValidateWord(guess []*Guess) bool {
	word := guessToWord(guess)
	_, ok := b.allowed[word]
	return ok
}

//go:embed data/words.json
var words []byte

//go:embed data/allowed.json
var allowed []byte

func NewStaticBackend() Backend {
	var wordList []string
	err := json.Unmarshal(words, &wordList)
	if err != nil {
		log.Fatalln(err)
	}

	var allowedList []string
	err = json.Unmarshal(allowed, &allowedList)
	if err != nil {
		log.Fatalln(err)
	}
	allowedList = append(allowedList, wordList...)

	allowedMap := make(map[string]bool)
	for i := range allowedList {
		allowedMap[allowedList[i]] = true
	}

	return StaticBackend{
		words:   wordList,
		allowed: allowedMap,
	}
}

func NewApiBackend() Backend {
	return ApiBackend{
		ApiBase: "https://www.wordsapi.com/mashape/words/",
		Auth:    "when=2022-01-26T16:38:02.746Z&encrypted=8cfdb188e722929be89707bee958bdb1aeb02f0937fb91b8",
	}
}

type ApiBackend struct {
	ApiBase string
	Auth    string
}

// https://wordsapiv1.p.mashape.com/words/?letters=6
/**
{"success":false,"message":"word not found"}
{"word":"test","definitions":[{"definition":"trying something to find out about it","partOfSpeech":"noun"},{"definition":"any standardized procedure for measuring sensitivity or memory or intelligence or aptitude or personality etc","partOfSpeech":"noun"},{"definition":"a set of questions or exercises evaluating skill or knowledge","partOfSpeech":"noun"},{"definition":"test or examine for the presence of disease or infection","partOfSpeech":"verb"},{"definition":"examine someone's knowledge of something","partOfSpeech":"verb"},{"definition":"put to the test, as for its quality, or give experimental use to","partOfSpeech":"verb"},{"definition":"the act of testing something","partOfSpeech":"noun"},{"definition":"the act of undergoing testing","partOfSpeech":"noun"},{"definition":"achieve a certain score or rating on a test","partOfSpeech":"verb"},{"definition":"a hard outer covering as of some amoebas and sea urchins","partOfSpeech":"noun"},{"definition":"determine the presence or properties of (a substance)","partOfSpeech":"verb"},{"definition":"show a certain characteristic when tested","partOfSpeech":"verb"},{"definition":"undergo a test","partOfSpeech":"verb"}]}
*/

func (b ApiBackend) apiRequest(path string) []byte {
	url := b.ApiBase + path
	if strings.Contains(url, "?") {
		url += "&"
	} else {
		url += "?"
	}
	url += b.Auth

	resp, err := http.Get(url)
	if err != nil {
		//TODO: better error handling
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//TODO: better error handling
		log.Fatalln(err)
	}
	return body
}

type ValidWordResponse struct {
	Word string `json:"word"`
}

func (b ApiBackend) GetRandomWord(length int) string {
	body := b.apiRequest(fmt.Sprintf("?letters=%d&random=true&partOfSpeech=noun", length))
	log.Printf("body: %s", body)

	var data ValidWordResponse
	err := json.Unmarshal(body, &data)
	if err != nil {
		//TODO: better error handling
		log.Fatalln(err)
	}
	return data.Word
}

func (b ApiBackend) ValidateWord(guess []*Guess) bool {
	word := guessToWord(guess)

	body := b.apiRequest(word + "/definitions")
	var data ValidWordResponse
	err := json.Unmarshal(body, &data)
	if err != nil {
		//TODO: better error handling
		log.Fatalln(err)
	}
	return data.Word != ""
}
