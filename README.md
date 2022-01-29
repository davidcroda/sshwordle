# SSHWordle
Terminal based [wordle](https://www.powerlanguage.co.uk/wordle/) clone. Uses the amazing [charm.sh](https://charm.sh)
libraries to render and expose the game over SSH. Stores  user statistics by IP. Supports 
pluggable backends. Currently includes a static file backend which uses the word lists from 
the original game, as well as an API based backend which uses the [WordsAPI.com](https://www.wordsapi.com/)
API to generate a random word as well as validate guesses.

## Usage
```shell
./sshwordle -h
Usage of ./sshwordle:
  -api
    	Use WordAPI to generate and verify words. If not specified uses hardcoded list of words from original Wordle game
  -host string
    	Host address for SSH server to listen (default "127.0.0.1")
  -port int
    	Port for SSH server to listen (default 1337)

```

## Building
```shell
make build
```
