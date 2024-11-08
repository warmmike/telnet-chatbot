package telsh

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"

	"bytes"
	"io"
	"strings"
	"sync"
)

type Config struct {
	APIIP string `json:"APIIP"`
	Port  string `json:"port"`
}

const (
	defaultExitCommandName = "bye"
	defaultPrompt          = ""
	defaultWelcomeMessage  = "\r\nWelcome!\r\n"
	defaultExitMessage     = "\r\nGoodbye!\r\n"
)

type ShellHandler struct {
	muxtex       sync.RWMutex
	producers    map[string]Producer
	elseProducer Producer

	ExitCommandName string
	Prompt          string
	WelcomeMessage  string
	ExitMessage     string
}

func NewShellHandler() *ShellHandler {
	producers := map[string]Producer{}

	telnetHandler := ShellHandler{
		producers: producers,

		Prompt:          defaultPrompt,
		ExitCommandName: defaultExitCommandName,
		WelcomeMessage:  defaultWelcomeMessage,
		ExitMessage:     defaultExitMessage,
	}

	return &telnetHandler
}

func (telnetHandler *ShellHandler) Register(name string, producer Producer) error {

	telnetHandler.muxtex.Lock()
	telnetHandler.producers[name] = producer
	telnetHandler.muxtex.Unlock()

	return nil
}

func (telnetHandler *ShellHandler) MustRegister(name string, producer Producer) *ShellHandler {
	if err := telnetHandler.Register(name, producer); nil != err {
		panic(err)
	}

	return telnetHandler
}

func (telnetHandler *ShellHandler) RegisterHandlerFunc(name string, handlerFunc HandlerFunc) error {

	produce := func(telctx telnet.Context, name string, args ...string) Handler {
		return PromoteHandlerFunc(handlerFunc, args...)
	}

	producer := ProducerFunc(produce)

	return telnetHandler.Register(name, producer)
}

func (telnetHandler *ShellHandler) MustRegisterHandlerFunc(name string, handlerFunc HandlerFunc) *ShellHandler {
	if err := telnetHandler.RegisterHandlerFunc(name, handlerFunc); nil != err {
		panic(err)
	}

	return telnetHandler
}

func (telnetHandler *ShellHandler) RegisterElse(producer Producer) error {

	telnetHandler.muxtex.Lock()
	telnetHandler.elseProducer = producer
	telnetHandler.muxtex.Unlock()

	return nil
}

func (telnetHandler *ShellHandler) MustRegisterElse(producer Producer) *ShellHandler {
	if err := telnetHandler.RegisterElse(producer); nil != err {
		panic(err)
	}

	return telnetHandler
}

func (telnetHandler *ShellHandler) ServeTELNET(telctx telnet.Context, writer telnet.Writer, reader telnet.Reader) {

	logger := telctx.Logger()
	if nil == logger {
		logger = internalDiscardLogger{}
	}

	var prompt bytes.Buffer
	var exitCommandName string
	//var welcomeMessage string
	var exitMessage string

	prompt.WriteString(telnetHandler.Prompt)

	promptBytes := prompt.Bytes()

	exitCommandName = telnetHandler.ExitCommandName
	//welcomeMessage = telnetHandler.WelcomeMessage
	exitMessage = telnetHandler.ExitMessage

	data, err := os.ReadFile("config.json")
	if err != nil {
		logger.Errorf("Error reading config file", err)
		return
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Errorf("Error parsing config file", err)
		return
	}

	baseURL := fmt.Sprintf("http://%s:%s/", config.APIIP, config.Port)
	client := openai.NewClient(
		option.WithBaseURL(baseURL),
	)

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are Joshua (formal designation: WOPR for War Operation Plan Response) is the main antagonist of the 1983 political sci-fi thriller film WarGames.  It is a super-computer system gifted with artificial intelligence, which was created and developed by the scientist Dr. Stephen Falken, who named it Joshua after his son who had died at an early age. It was developed by Stephen to become a true artificial intelligence, learning by continually engaging in both traditional games and highly complex combat simulations (the titular war games.) After a chance encounter with a civilian, David Lightwood, who believes the games to be mere entertainment, Joshua/WOPR continues to treat the war scenario as an actual war that it must prevail in. As it cannot tell the simulation from reality, Joshua becomes intent on igniting World War III in the real world. Nuclear war. A strange game, the only winning move is not to play. How about a nice game of chess?"),
			openai.UserMessage("Can you present 5 lines of random ascii characters like you are a computer from the 1980s fritzing out."),
			openai.AssistantMessage("Just present the ascii and nothing else, but really make it random."),
		}),
		Seed:  openai.Int(1),
		Model: openai.F(openai.ChatModelGPT4o),
	}
	ctx := context.Background()
	stream := client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}
	oi.LongWriteString(writer, "\r\n\r\n")
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		if len(chunk.Choices) > 0 {
			if strings.ContainsAny(chunk.Choices[0].Delta.Content, "\n") {
				chunk.Choices[0].Delta.Content = strings.Replace(chunk.Choices[0].Delta.Content, "\n", "\r\n", -1)
			}
			oi.LongWriteString(writer, chunk.Choices[0].Delta.Content)
		}
	}

	params.Messages.Value = append(params.Messages.Value, acc.Choices[0].Message)

	params.Messages.Value = append(params.Messages.Value, openai.UserMessage("Say only 'GREETINGS PROFESSOR FALCON' and nothing more."))
	stream = client.Chat.Completions.NewStreaming(ctx, params)
	acc = openai.ChatCompletionAccumulator{}
	oi.LongWriteString(writer, "\r\n\r\n")

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		if len(chunk.Choices) > 0 {
			oi.LongWriteString(writer, chunk.Choices[0].Delta.Content)
		}
	}
	params.Messages.Value = append(params.Messages.Value, acc.Choices[0].Message)

	params.Messages.Value = append(params.Messages.Value, openai.UserMessage("Say 'SHALL WE PLAY A GAME?' and nothing more"))
	stream = client.Chat.Completions.NewStreaming(ctx, params)
	acc = openai.ChatCompletionAccumulator{}
	oi.LongWriteString(writer, "\r\n\r\n")

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		if len(chunk.Choices) > 0 {
			oi.LongWriteString(writer, chunk.Choices[0].Delta.Content)
		}
	}
	oi.LongWriteString(writer, "\r\n\r\n")
	params.Messages.Value = append(params.Messages.Value, acc.Choices[0].Message)

	if _, err := oi.LongWrite(writer, promptBytes); nil != err {
		logger.Errorf("Problem long writing prompt: %v", err)
		return
	}
	logger.Debugf("Wrote prompt: %q.", promptBytes)

	var buffer [1]byte // Seems like the length of the buffer needs to be small, otherwise will have to wait for buffer to fill up.
	p := buffer[:]

	var line bytes.Buffer

	var lineString string
	charcount := 0
	for {
		// Read 1 byte.
		n, err := reader.Read(p)
		if n <= 0 && nil == err {
			continue
		} else if n <= 0 && nil != err {
			break
		}

		//line.WriteByte(p[0])
		//logger.Tracef("Received: %q (%d).", p[0], p[0])
		if p[0] != 8 {
			if p[0] != 27 {
				charcount += 1
				oi.LongWrite(writer, p[:n])
				line.WriteByte(p[0])
				lineString = line.String()
				//logger.Tracef("Received: %q (%d).", p[0], p[0])
			}
		} else {
			if charcount > 0 {
				charcount -= 1
				oi.LongWrite(writer, []byte("\b \b"))
				line.Truncate(len(lineString) - 1)
				lineString = line.String()
			}
		}

		if '\n' == p[0] {
			lineString := line.String()

			if "\r\n" == lineString {
				line.Reset()
				if _, err := oi.LongWrite(writer, promptBytes); nil != err {
					return
				}
				continue
			}

			//@TODO: support piping.
			fields := strings.Fields(lineString)
			logger.Debugf("Have %d tokens.", len(fields))
			logger.Tracef("Tokens: %v", fields)
			if len(fields) <= 0 {
				line.Reset()
				if _, err := oi.LongWrite(writer, promptBytes); nil != err {
					return
				}
				continue
			}

			field0 := fields[0]

			params.Messages.Value = append(params.Messages.Value, openai.UserMessage(lineString))
			stream = client.Chat.Completions.NewStreaming(ctx, params)
			acc = openai.ChatCompletionAccumulator{}
			oi.LongWriteString(writer, "\r\n")
			for stream.Next() {
				chunk := stream.Current()
				acc.AddChunk(chunk)
				if len(chunk.Choices) > 0 {
					if strings.ContainsAny(chunk.Choices[0].Delta.Content, "\n") {
						chunk.Choices[0].Delta.Content = strings.Replace(chunk.Choices[0].Delta.Content, "\n", "\r\n", -1)
					}
					oi.LongWriteString(writer, chunk.Choices[0].Delta.Content)
				}
			}
			oi.LongWriteString(writer, "\r\n\r\n")
			params.Messages.Value = append(params.Messages.Value, acc.Choices[0].Message)

			if strings.ToLower(exitCommandName) == strings.ToLower(field0) {
				oi.LongWriteString(writer, exitMessage)
				return
			}

			line.Reset()
			if _, err := oi.LongWrite(writer, promptBytes); nil != err {
				return
			}
		}

		//@TODO: Are there any special errors we should be dealing with separately?
		if nil != err {
			break
		}
	}

	oi.LongWriteString(writer, exitMessage)
	return
}

func connect(telctx telnet.Context, writer io.Writer, reader io.Reader) {

	logger := telctx.Logger()

	go func(logger telnet.Logger) {

		var buffer [1]byte // Seems like the length of the buffer needs to be small, otherwise will have to wait for buffer to fill up.
		p := buffer[:]

		for {
			// Read 1 byte.
			n, err := reader.Read(p)
			if n <= 0 && nil == err {
				continue
			} else if n <= 0 && nil != err {
				break
			}

			//logger.Tracef("Sending: %q.", p)
			oi.LongWrite(writer, p)
			//logger.Tracef("Sent: %q.", p)
		}
	}(logger)
}
