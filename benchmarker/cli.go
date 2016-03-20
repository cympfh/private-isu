package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/score"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota

	FailThreshold           = 5
	InitializeTimeout       = time.Duration(10) * time.Second
	BenchmarkTimeout        = 30 * time.Second
	DetailedCheckQueueSize  = 2
	NonNormalCheckQueueSize = 2
	WaitAfterTimeout        = 5

	PostsPerPage = 20
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

type user struct {
	AccountName string
	Password    string
}

type Output struct {
	Pass     bool     `json:"pass"`
	Score    int64    `json:"score"`
	Suceess  int64    `json:"success"`
	Fail     int64    `json:"fail"`
	Messages []string `json:"messages"`
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		target   string
		userdata string

		version bool
		debug   bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&target, "target", "", "")
	flags.StringVar(&target, "t", "", "(Short)")

	flags.StringVar(&userdata, "userdata", "", "userdata directory")
	flags.StringVar(&userdata, "u", "", "userdata directory")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	flags.BoolVar(&debug, "debug", false, "Debug mode")
	flags.BoolVar(&debug, "d", false, "Debug mode")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	targetHost, terr := checker.SetTargetHost(target)
	if terr != nil {
		outputNeedToContactUs(terr.Error())
		return ExitCodeError
	}

	initialize := make(chan bool)

	setupInitialize(targetHost, initialize)

	users, _, adminUsers, sentences, images, err := prepareUserdata(userdata)
	if err != nil {
		outputNeedToContactUs(err.Error())
		return ExitCodeError
	}

	initReq := <-initialize

	if !initReq {
		fmt.Println(outputResultJson(false, []string{"初期化リクエストに失敗しました"}))

		return ExitCodeError
	}

	// 最初にDOMチェックなどをやってしまい、通らなければさっさと失敗させる
	commentScenario(checker.NewSession(), randomUser(users), randomUser(users).AccountName, randomSentence(sentences))
	postImageScenario(checker.NewSession(), randomUser(users), randomImage(images), randomSentence(sentences))
	cannotLoginNonexistentUserScenario(checker.NewSession())
	cannotLoginWrongPasswordScenario(checker.NewSession(), randomUser(users))
	cannotAccessAdminScenario(checker.NewSession(), randomUser(users))
	cannotPostWrongCSRFTokenScenario(checker.NewSession(), randomUser(users), randomImage(images))
	loginScenario(checker.NewSession(), randomUser(users))
	banScenario(checker.NewSession(), checker.NewSession(), randomUser(users), randomUser(adminUsers), randomImage(images), randomSentence(sentences))

	if score.GetInstance().GetFails() > 0 {
		msgs := []string{}
		for _, err := range score.GetFailErrors() {
			msgs = append(msgs, fmt.Sprint(err.Error()))
		}
		fmt.Println(outputResultJson(false, msgs))
		return ExitCodeError
	}

	indexMoreAndMoreScenarioCh := makeChanBool(2)
	loadIndexScenarioCh := makeChanBool(2)
	userAndPostPageScenarioCh := makeChanBool(2)
	commentScenarioCh := makeChanBool(1)
	postImageScenarioCh := makeChanBool(1)
	loginScenarioCh := makeChanBool(1)
	banScenarioCh := makeChanBool(1)
	nonNormalCheckCh := makeChanBool(NonNormalCheckQueueSize)

	timeoutCh := time.After(BenchmarkTimeout)

L:
	for {
		select {
		case <-indexMoreAndMoreScenarioCh:
			go func() {
				indexMoreAndMoreScenario(checker.NewSession())
				indexMoreAndMoreScenarioCh <- true
			}()
		case <-loadIndexScenarioCh:
			go func() {
				loadIndexScenario(checker.NewSession())
				loadIndexScenarioCh <- true
			}()
		case <-userAndPostPageScenarioCh:
			go func() {
				userAndPostPageScenario(checker.NewSession(), randomUser(users).AccountName)
				userAndPostPageScenarioCh <- true
			}()
		case <-commentScenarioCh:
			go func() {
				commentScenario(checker.NewSession(), randomUser(users), randomUser(users).AccountName, randomSentence(sentences))
				commentScenarioCh <- true
			}()
		case <-postImageScenarioCh:
			go func() {
				postImageScenario(checker.NewSession(), randomUser(users), randomImage(images), randomSentence(sentences))
				postImageScenarioCh <- true
			}()
		case <-nonNormalCheckCh:
			go func() {
				cannotLoginNonexistentUserScenario(checker.NewSession())
				cannotLoginWrongPasswordScenario(checker.NewSession(), randomUser(users))
				cannotAccessAdminScenario(checker.NewSession(), randomUser(users))
				cannotPostWrongCSRFTokenScenario(checker.NewSession(), randomUser(users), randomImage(images))
				<-time.After(3 * time.Second)
				nonNormalCheckCh <- true
			}()
		case <-loginScenarioCh:
			go func() {
				loginScenario(checker.NewSession(), randomUser(users))
				loginScenarioCh <- true
			}()
		case <-banScenarioCh:
			go func() {
				banScenario(checker.NewSession(), checker.NewSession(), randomUser(users), randomUser(adminUsers), randomImage(images), randomSentence(sentences))
				banScenarioCh <- true
			}()
		case <-timeoutCh:
			break L
		}
	}

	time.Sleep(WaitAfterTimeout)

	msgs := []string{}

	if !debug {
		// 通常は適当にsortしてuniqしたログを出す
		for _, err := range score.GetFailErrors() {
			msgs = append(msgs, fmt.Sprint(err.Error()))
		}
	} else {
		// debugモードなら生ログを出力
		for _, err := range score.GetFailRawErrors() {
			msgs = append(msgs, fmt.Sprint(err.Error()))
		}
	}

	fmt.Println(outputResultJson(true, msgs))

	return ExitCodeOK
}

func outputResultJson(pass bool, messages []string) string {
	output := Output{
		Pass:     pass,
		Score:    score.GetInstance().GetScore(),
		Suceess:  score.GetInstance().GetSucesses(),
		Fail:     score.GetInstance().GetFails(),
		Messages: messages,
	}

	b, _ := json.Marshal(output)

	return string(b)
}

// 主催者に連絡して欲しいエラー
func outputNeedToContactUs(message string) {
	outputResultJson(false, []string{"！！！主催者に連絡してください！！！", message})
}

func makeChanBool(len int) chan bool {
	ch := make(chan bool, len)
	for i := 0; i < len; i++ {
		ch <- true
	}
	return ch
}

func randomUser(users []user) user {
	return users[util.RandomNumber(len(users))]
}

func randomImage(images []*checker.Asset) *checker.Asset {
	return images[util.RandomNumber(len(images))]
}

func randomSentence(sentences []string) string {
	return sentences[util.RandomNumber(len(sentences))]
}

func setupInitialize(targetHost string, initialize chan bool) {
	go func(targetHost string) {
		client := &http.Client{
			Timeout: InitializeTimeout,
		}

		parsedURL, _ := url.Parse("/initialize")
		parsedURL.Scheme = "http"
		parsedURL.Host = targetHost

		res, err := client.Get(parsedURL.String())
		if err != nil {
			initialize <- false
			return
		}
		defer res.Body.Close()
		initialize <- true
	}(targetHost)
}
