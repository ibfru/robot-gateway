package main

import (
	"bytes"
	"community-robot-lib/config"
	"community-robot-lib/framework"
	"community-robot-lib/utils"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"sync"
)

const botName = "robot-atomgit-access"

func newRobot() *robot {
	return &robot{hc: utils.NewHttpClient(3)}
}

type robot struct {
	// ec is an http client used for dispatching events
	// to external plugin services.
	hc *utils.HttpClient
	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) getConfig(cfg config.Config) (*configuration, error) {
	if c, ok := cfg.(*configuration); ok {
		return c, nil
	}
	return nil, errors.New("can't convert to configuration")
}

func (bot *robot) RegisterEventHandler(f framework.HandlerRegister) {
	f.RegisterPreEventHandler(bot.handleRequest)
	f.RegisterAccessHandler(bot.handleAccessEvent)
}

type ResJson struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Event   *framework.GenericEvent `json:"event"`
}

func (bot *robot) handleRequest(w http.ResponseWriter, r *http.Request) *framework.GenericEvent {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Warn("when webhook body close, error occurred:", err)
		}
	}(r.Body)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("when webhook body to be read, error occurred:", err)
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8890/1.json", bytes.NewBuffer(body))
	if err != nil {
		logrus.Error("when webhook body to be read, error occurred:", err)
		return nil
	}
	req.Header = r.Header
	var resBody ResJson
	resStatusCode, err := bot.hc.DoWait(req, &resBody)
	if err != nil {
		return nil
	}

	if resStatusCode != http.StatusOK {
		http.Error(w, resBody.Message, resStatusCode)
		logrus.Error("service occurred error")
		return nil
	}

	if resBody.Code != http.StatusOK {
		http.Error(w, resBody.Message, 400)
		logrus.Error("webhook request check error")
		return nil
	}

	ge := resBody.Event
	ge.SourcePayload = body
	return ge
}

func (bot *robot) handleAccessEvent(evt *framework.GenericEvent, cnf config.Config, lgr *logrus.Entry) error {
	c, ok := cnf.(*configuration)
	if !ok {
		return fmt.Errorf("can't convert to configuration")
	}

	endpoints := c.GetEndpoints(evt.Org, evt.Repo, evt.EventName)
	bot.dispatchToDownstreamRobot(endpoints, lgr, evt)
	return nil
}

func (bot *robot) dispatchToDownstreamRobot(endpoints []string, lgr *logrus.Entry, evt *framework.GenericEvent) {

	newReq := func(endpoint string) (*http.Request, error) {
		payload, err := evt.ConvertToBytes()
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(payload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("token", "111")
		return req, nil
	}

	reqSize := len(endpoints)
	for i := 0; i < reqSize; i++ {
		if r, err := newReq(endpoints[i]); err == nil {
			bot.wg.Add(1)
			go func(req *http.Request) {
				_, _ = bot.hc.DoSend(req)
				bot.wg.Done()
				fmt.Println("================>")
			}(r)
		} else {
			lgr.WithField("endpoint", endpoints[i]).Error("Error generating http request.", err)
		}
	}
}
