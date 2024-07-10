package main

import (
	"flag"
	_ "net/http"
	_ "strconv"
	"time"

	_ "community-robot-lib/config"
	"community-robot-lib/framework"
	_ "community-robot-lib/interrupts"
	"community-robot-lib/logrusutil"
	liboptions "community-robot-lib/options"
	"community-robot-lib/secret"
	_ "community-robot-lib/utils"
	"github.com/sirupsen/logrus"
)

type options struct {
	service liboptions.ServiceOptions
	client  liboptions.ClientOptions
}

func (o *options) Validate() error {
	if err := o.service.Validate(); err != nil {
		return err
	}

	return o.client.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var opt options

	opt.client.AddFlags(fs)
	opt.service.AddFlags(fs)

	_ = fs.Parse(args)

	return opt
}

func main() {
	logrusutil.ComponentInit(botName)

	//o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	opt := options{
		service: liboptions.ServiceOptions{
			Port:        8822,
			ConfigFile:  "D:\\Project\\github\\ibfru\\robot-gateway\\local\\config.yaml",
			GracePeriod: 300 * time.Second,
		},
		client: liboptions.ClientOptions{
			TokenPath:   "D:\\Project\\github\\ibfru\\robot-gateway\\local\\secret",
			HandlerPath: "/atomgit-hook",
		},
	}

	if err := opt.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	secretAgent := new(secret.Agent)
	if err := secretAgent.Start([]string{opt.client.TokenPath}); err != nil {
		logrus.WithError(err).Fatal("Error starting secret agent.")
	}
	defer secretAgent.Stop()

	// to replace

	p := newRobot()
	opt.client.TokenGenerator = secretAgent.GetTokenGenerator(opt.client.TokenPath)
	framework.Run(p, opt.service, opt.client)
}
