package framework

import (
	"community-robot-lib/config"
	sdk "git-platform-sdk"
	"github.com/sirupsen/logrus"
)

type GenericHandler func(e *sdk.GenericEvent, cfg config.Config, log *logrus.Entry) error

type handlers struct {
	accessHandlers            GenericHandler
	issueHandlers             GenericHandler
	pullRequestHandler        GenericHandler
	pushEventHandler          GenericHandler
	issueCommentHandler       GenericHandler
	reviewEventHandler        GenericHandler
	reviewCommentEventHandler GenericHandler
}

// RegisterAccessHandler registers a plugin's AnyEvent handler.
func (h *handlers) RegisterAccessHandler(fn GenericHandler) {
	h.accessHandlers = fn
}

// RegisterIssueHandler registers a plugin's IssueEvent handler.
func (h *handlers) RegisterIssueHandler(fn GenericHandler) {
	h.issueHandlers = fn
}

// RegisterPullRequestHandler registers a plugin's PullRequestEvent handler.
func (h *handlers) RegisterPullRequestHandler(fn GenericHandler) {
	h.pullRequestHandler = fn
}

// RegisterPushEventHandler registers a plugin's PushEvent handler.
func (h *handlers) RegisterPushEventHandler(fn GenericHandler) {
	h.pushEventHandler = fn
}

// RegisterIssueCommentHandler registers a plugin's IssueCommentEvent handler.
func (h *handlers) RegisterIssueCommentHandler(fn GenericHandler) {
	h.issueCommentHandler = fn
}

// RegisterReviewEventHandler registers a plugin's ReviewEvent handler.
func (h *handlers) RegisterReviewEventHandler(fn GenericHandler) {
	h.reviewEventHandler = fn
}

// RegisterReviewCommentEventHandler registers a plugin's ReviewCommentEvent handler.
func (h *handlers) RegisterReviewCommentEventHandler(fn GenericHandler) {
	h.reviewCommentEventHandler = fn
}
