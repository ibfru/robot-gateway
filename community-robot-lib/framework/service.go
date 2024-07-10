package framework

import (
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"

	"community-robot-lib/config"
	"community-robot-lib/interrupts"
	"community-robot-lib/options"
)

type HandlerRegister interface {
	RegisterAccessHandler(GenericHandler)
	RegisterIssueHandler(GenericHandler)
	RegisterPullRequestHandler(GenericHandler)
	RegisterPushEventHandler(GenericHandler)
	RegisterIssueCommentHandler(GenericHandler)
	RegisterReviewEventHandler(GenericHandler)
	RegisterReviewCommentEventHandler(GenericHandler)
}

type Robot interface {
	NewConfig() config.Config
	RegisterEventHandler(HandlerRegister)
}

func Run(bot Robot, servOpt options.ServiceOptions, clientOpt options.ClientOptions) {
	agent := config.NewConfigAgent(bot.NewConfig)
	if err := agent.Start(servOpt.ConfigFile); err != nil {
		logrus.WithError(err).Errorf("start config:%s", servOpt.ConfigFile)
		return
	}

	h := handlers{}
	bot.RegisterEventHandler(&h)

	d := &dispatcher{agent: &agent, h: h, hmac: clientOpt.TokenGenerator}
	GetClientInstance(d)

	defer interrupts.WaitForGracefulShutdown()

	interrupts.OnInterrupt(func() {
		agent.Stop()
		d.Wait()
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// service's healthy check, do nothing
	})

	http.Handle(clientOpt.HandlerPath, d)

	httpServer := &http.Server{Addr: ":" + strconv.Itoa(servOpt.Port)}

	interrupts.ListenAndServe(httpServer, servOpt.GracePeriod)
}
