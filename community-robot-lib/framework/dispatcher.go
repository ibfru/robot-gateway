package framework

import (
	"io"
	"net/http"
	"sync"

	"community-robot-lib/config"
	sdk "git-platform-sdk"
	"github.com/sirupsen/logrus"
)

const (
	LogFieldOrg       = "org"
	LogFieldRepo      = "repo"
	LogFieldEventId   = "event_id"
	LogFieldEventType = "event-type"
	LogFieldPayload   = "payload"
	logFieldURL       = "url"
	logFieldAction    = "action"

	UserAgentHeader = "Robot-Gateway-Access"

	badRequestMessagePrefix = "400 Bad Request: "
)

type dispatcher struct {
	agent *config.ConfigAgent

	h handlers

	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup

	// secret usage
	hmac func() []byte
}

func (d *dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ge := parseRequest(w, r, d.hmac)
	if ge == nil {
		return
	}

	l := logrus.WithFields(ge.ConvertToMap())

	if err := d.Dispatch(ge, l); err != nil {
		l.WithError(err).Error()
	}
}

func (d *dispatcher) Dispatch(event *sdk.GenericEvent, l *logrus.Entry) error {
	if event.EventType < AccessEvent || event.EventType > PullRequestCommentEvent {
		l.Debug("Ignoring unknown event type")
	} else {

		d.wg.Add(1)
		go d.handleEvent(event, l)
	}

	return nil
}

func (d *dispatcher) Wait() {
	d.wg.Wait() // Handle remaining requests
}

var handlerList []GenericHandler
var once sync.Once

// Event-Type Value
const (
	AccessEvent = iota
	IssueEvent
	PullRequestEvent
	PushEvent
	IssueCommentEvent
	PullRequestReviewEvent
	PullRequestCommentEvent
)

func (d *dispatcher) initialClient() {
	handlerList = []GenericHandler{
		d.h.accessHandlers,
		d.h.issueHandlers,
		d.h.pullRequestHandler,
		d.h.pushEventHandler,
		d.h.issueCommentHandler,
		d.h.reviewEventHandler,
		d.h.reviewCommentEventHandler,
	}
}

func GetClientInstance(d *dispatcher) *[]GenericHandler {
	once.Do(d.initialClient)
	return &handlerList
}

func (d *dispatcher) getConfig() config.Config {
	_, c := d.agent.GetConfig()

	return c
}

// handleAccessEvent access robot handle request that come form webhook
func (d *dispatcher) handleEvent(e *sdk.GenericEvent, l *logrus.Entry) {
	defer d.wg.Done()

	fn := (*GetClientInstance(d))[e.EventType]
	if err := fn(e, d.getConfig(), l); err != nil {
		l.WithError(err).Error()
	} else {
		l.Info()
	}
}

func parseRequest(w http.ResponseWriter, r *http.Request, getHmac func() []byte) *sdk.GenericEvent {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Warn("when webhook body close, error occurred:", err)
		}
	}(r.Body)

	resp := func(code int, msg string) {
		http.Error(w, msg, code)
	}
	ge := sdk.GenericEvent{}
	var err error
	ge.EventType, ge.EventName, err = sdk.GetEventType(&r.Header)
	if err != nil {
		resp(http.StatusBadRequest, badRequestMessagePrefix+err.Error())
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		resp(http.StatusInternalServerError, "500 Internal Server Error: Failed to read request body")
		return nil
	}

	if ge.EventType == AccessEvent {

		if ge.EventUUID, err = sdk.GetEventUUID(&r.Header); err != nil {
			resp(http.StatusBadRequest, badRequestMessagePrefix+err.Error())
			return nil
		}

		if err = sdk.AuthSign(&r.Header, &body, getHmac()); err != nil {
			resp(http.StatusForbidden, "403 Forbidden: "+err.Error())
			return nil
		}

		resp(http.StatusOK, "The request was accepted by access's robot, inform to webhook.")

	} else {
		if err = sdk.CheckUserAgent(&r.Header, UserAgentHeader); err != nil {
			resp(http.StatusBadRequest, badRequestMessagePrefix+err.Error())
			return nil
		}
	}

	ge.Payload = body
	return &ge
}
