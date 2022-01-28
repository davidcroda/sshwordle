package main

import (
	"flag"
	"sshwordle/src/sshwordle"
)

func main() {
	var apiFlag = flag.Bool("api", false,
		"Use WordAPI to generate and verify words. "+
			"If not specified uses hardcoded list of words "+
			"from original Wordle game")
	var host = flag.String("host", "127.0.0.1", "Host address for SSH server to listen")
	var port = flag.Int("port", 1337, "Port for SSH server to listen")

	flag.Parse()

	sshwordle.StartServer(*host, *port, *apiFlag)
}
