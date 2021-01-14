package main

import (
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/informer"
	"github.com/xjayleex/kauloud/pkg/utils"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	path, err := utils.DefaultKubeConfigPath()
	if err != nil {
		panic(err)
	}
	clientset, err := utils.GetKubeClientFromConfigPath(path)
	if err != nil {
		panic(err)
	}

	describer := informer.NewPollingResourceDescriber(clientset, logger)
	stop := make(chan struct{})
	defer close(stop)
	describer.Run(1, stop)
	select {}
}
/*
func init () {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}


func main () {
	app := &cli.App{
		UseShortOptionHandling: true,
		Name: "kauloud",
		Usage: "To control kau kube cloud.",
		Commands: []*cli.Command{
			&RunServer,
			&StopServer,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
*/