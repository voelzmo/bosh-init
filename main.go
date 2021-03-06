package main

import (
	"os"
	"path"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshlogfile "github.com/cloudfoundry/bosh-agent/logger/file"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
	boshtime "github.com/cloudfoundry/bosh-agent/time"
	boshuuid "github.com/cloudfoundry/bosh-agent/uuid"

	bicmd "github.com/cloudfoundry/bosh-init/cmd"

	biui "github.com/cloudfoundry/bosh-init/ui"
	biuifmt "github.com/cloudfoundry/bosh-init/ui/fmt"
)

const mainLogTag = "main"

func main() {
	logger := newLogger()
	defer logger.HandlePanic("Main")
	fileSystem := boshsys.NewOsFileSystem(logger)
	workspaceRootPath := path.Join(os.Getenv("HOME"), ".bosh_init")
	ui := biui.NewConsoleUI(logger)

	timeService := boshtime.NewConcreteService()

	cmdFactory := bicmd.NewFactory(
		fileSystem,
		ui,
		timeService,
		logger,
		boshuuid.NewGenerator(),
		workspaceRootPath,
	)

	cmdRunner := bicmd.NewRunner(cmdFactory)
	stage := biui.NewStage(ui, timeService, logger)
	err := cmdRunner.Run(stage, os.Args[1:]...)
	if err != nil {
		displayHelpFunc := func() {
			if strings.Contains(err.Error(), "Invalid usage") {
				ui.ErrorLinef("")
				cmdRunner.Run(stage, append([]string{"help"}, os.Args[1:]...)...)
			}
		}
		fail(err, ui, logger, displayHelpFunc)
	}
}

func newLogger() boshlog.Logger {
	logLevelString := os.Getenv("BOSH_INIT_LOG_LEVEL")
	level := boshlog.LevelNone
	if logLevelString != "" {
		var err error
		level, err = boshlog.Levelify(logLevelString)
		if err != nil {
			err = bosherr.WrapError(err, "Invalid BOSH_INIT_LOG_LEVEL value")
			logger := boshlog.NewLogger(boshlog.LevelError)
			ui := biui.NewConsoleUI(logger)
			fail(err, ui, logger, nil)
		}
	}

	logPath := os.Getenv("BOSH_INIT_LOG_PATH")
	if logPath != "" {
		return newFileLogger(logPath, level)
	}

	return boshlog.NewLogger(level)
}

func newFileLogger(logPath string, level boshlog.LogLevel) boshlog.Logger {
	// Log file logger errors to the STDERR logger
	logger := boshlog.NewLogger(boshlog.LevelError)
	fileSystem := boshsys.NewOsFileSystem(logger)

	// log file will be closed by process exit
	// log file readable by all
	logger, _, err := boshlogfile.New(level, logPath, boshlogfile.DefaultLogFileMode, fileSystem)
	if err != nil {
		logger := boshlog.NewLogger(boshlog.LevelError)
		ui := biui.NewConsoleUI(logger)
		fail(err, ui, logger, nil)
	}
	return logger
}

func fail(err error, ui biui.UI, logger boshlog.Logger, callback func()) {
	logger.Error(mainLogTag, err.Error())
	ui.ErrorLinef("")
	ui.ErrorLinef(biuifmt.MultilineError(err))
	if callback != nil {
		callback()
	}
	os.Exit(1)
}
