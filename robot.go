package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"community-robot-lib/config"
	"community-robot-lib/framework"
	sdk "git-platform-sdk"
	"github.com/sirupsen/logrus"
)

const botName = "robot-atomgit-access"

type iClient interface {
}

func newRobot() *robot {
	return &robot{}
}

type robot struct {
	cli iClient
}

type accessDispatcher struct {
	// ec is an http client used for dispatching events
	// to external plugin services.
	ec http.Client
	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup
}

var ad = accessDispatcher{
	ec: *http.DefaultClient,
	wg: sync.WaitGroup{},
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
	f.RegisterAccessHandler(bot.handleAccessEvent)
}

func (bot *robot) handleAccessEvent(e *sdk.GenericEvent, cfg config.Config, log *logrus.Entry) error {
	c, ok := cfg.(*configuration)
	if !ok {
		return fmt.Errorf("can't convert to configuration")
	}

	endpoints := c.GetEndpoints(e.Org, e.Repo, e.EventName)
	ad.dispatchToDownstreamRobot(endpoints, log, e)
	return nil
}

func (d *accessDispatcher) dispatchToDownstreamRobot(endpoints []string, l *logrus.Entry, e *sdk.GenericEvent) {

	newReq := func(endpoint string) (*http.Request, error) {
		payload, err := e.ConvertToBytes()
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(payload))
		if err != nil {
			return nil, err
		}

		req.Header.Set(sdk.WebhookUserAgentKey, framework.UserAgentHeader)
		req.Header.Set("token", "111")
		return req, nil
	}

	reqs := make([]*http.Request, 0, len(endpoints))

	for _, endpoint := range endpoints {
		if req, err := newReq(endpoint); err == nil {
			reqs = append(reqs, req)
		} else {
			l.WithError(err).WithField("endpoint", endpoint).Error("Error generating http request.")
		}
	}

	for _, req := range reqs {
		d.wg.Add(1)

		// concurrent action is sending request not generating it.
		// so, generates requests first.
		go func(req *http.Request) {
			defer d.wg.Done()

			if err := d.forwardTo(req); err != nil {
				l.WithError(err).WithField("endpoint", req.URL.String()).Error("Error forwarding event.")
			}
		}(req)
	}
}

func (d *accessDispatcher) forwardTo(req *http.Request) error {
	resp, err := d.do(req)
	if err != nil || resp == nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		e := Body.Close()
		if e != nil {
			logrus.Warn("when access received downstream response body close, error occurred:", e)
		}
	}(resp.Body)

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("response has status %q and body %q", resp.Status, string(rb))
	}
	return nil
}

func (d *accessDispatcher) do(req *http.Request) (resp *http.Response, err error) {
	if resp, err = d.ec.Do(req); err == nil {
		return
	}

	maxRetries := 4
	backoff := 100 * time.Millisecond

	for retries := 0; retries < maxRetries; retries++ {
		time.Sleep(backoff)
		backoff *= 2

		if resp, err = d.ec.Do(req); err == nil {
			break
		}
	}
	return
}
