package main

import (
	telsh "telnet-chatbot/telchat"

	"github.com/reiver/go-telnet"
)

func main() {

	shellHandler := telsh.NewShellHandler()

	shellHandler.WelcomeMessage = ""

	addr := ":5556"
	if err := telnet.ListenAndServe(addr, shellHandler); nil != err {
		panic(err)
	}
}
