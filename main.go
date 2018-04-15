package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/keybase/go-keybase-chat-bot/kbchat"
)

func main() {
	rc := mainInner()
	os.Exit(rc)
}

type Options struct {
	KeybaseLocation string
	ListenPort      int
	Channel         string
}

type BotServer struct {
	opts Options
	kbc  *kbchat.API
}

func NewBotServer(opts Options) *BotServer {
	return &BotServer{
		opts: opts,
	}
}

func (s *BotServer) debug(msg string, args ...interface{}) {
	fmt.Printf("BotServer: "+msg+"\n", args...)
}

type alert struct {
	Team    string
	Type    string `json:"alerttype"`
	Host    string
	Message string
	Hits    int `json:"num_hits"`

	RawSeverity     string `json:"severity"`
	RawTimestamp    string `json:"time"`
	RawProgram      string `json:"program"`
	SyslogSeverity  string `json:"syslog_severity"`
	SyslogTimestamp string `json:"syslog_timestamp"`
	SyslogProgram   string `json:"syslog_program"`
}

func (a alert) Severity() string {
	if a.RawSeverity == "" {
		return a.SyslogSeverity
	}
	return a.RawSeverity
}

func (a alert) Timestamp() string {
	if a.RawTimestamp == "" {
		return a.SyslogTimestamp
	}
	return a.RawTimestamp
}

func (a alert) Program() string {
	if a.RawProgram == "" {
		return a.SyslogProgram
	}
	return a.RawProgram
}

func (a alert) String() string {
	return fmt.Sprintf("*%s*\n>Severity: %s\n>Program: %s\n>Host: %s\n>Hits: %d\n>Timestamp: %s\n>Message: %s",
		a.Type, a.Severity(), a.Program(), a.Host, a.Hits, a.Timestamp(), a.Message)
}

func (s *BotServer) handlePost(w http.ResponseWriter, r *http.Request) {
	var a alert
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&a); err != nil {
		s.debug("failed to decode alert JSON: %s", err.Error())
		return
	}

	if err := s.kbc.SendMessageByTeamName(a.Team, a.String(), &s.opts.Channel); err != nil {
		s.debug("failed to send message: %s", err.Error())
	}
}

func (s *BotServer) Start() (err error) {

	// Start up KB chat
	if s.kbc, err = kbchat.Start(s.opts.KeybaseLocation); err != nil {
		return err
	}

	// Start up HTTP interface
	http.HandleFunc("/", s.handlePost)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.opts.ListenPort), nil)
}

func mainInner() int {
	var opts Options

	flag.StringVar(&opts.KeybaseLocation, "keybase", "keybase", "keybase command")
	flag.StringVar(&opts.Channel, "channel", "alerts", "channel to send messages")
	flag.IntVar(&opts.ListenPort, "port", 8080, "listen port")
	flag.Parse()

	bs := NewBotServer(opts)
	if err := bs.Start(); err != nil {
		fmt.Printf("error running chat loop: %s\n", err.Error())
	}

	return 0
}
