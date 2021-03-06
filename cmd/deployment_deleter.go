package cmd

import (
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
	biblobstore "github.com/cloudfoundry/bosh-init/blobstore"
	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	biconfig "github.com/cloudfoundry/bosh-init/config"
	bicpirel "github.com/cloudfoundry/bosh-init/cpi/release"
	bidepl "github.com/cloudfoundry/bosh-init/deployment"
	bihttpagent "github.com/cloudfoundry/bosh-init/deployment/agentclient/http"
	biinstall "github.com/cloudfoundry/bosh-init/installation"
	biinstallmanifest "github.com/cloudfoundry/bosh-init/installation/manifest"
	bitarball "github.com/cloudfoundry/bosh-init/installation/tarball"
	birel "github.com/cloudfoundry/bosh-init/release"
	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	biui "github.com/cloudfoundry/bosh-init/ui"
)

func NewDeploymentDeleter(
	ui biui.UI,
	logTag string,
	logger boshlog.Logger,
	fs boshsys.FileSystem,
	deploymentStateService biconfig.DeploymentStateService,
	releaseManager birel.Manager,
	installerFactory biinstall.InstallerFactory,
	cloudFactory bicloud.Factory,
	agentClientFactory bihttpagent.AgentClientFactory,
	blobstoreFactory biblobstore.Factory,
	deploymentManagerFactory bidepl.ManagerFactory,
	releaseSetParser birelsetmanifest.Parser,
	releaseSetValidator birelsetmanifest.Validator,
	releaseExtractor birel.Extractor,
	installationParser biinstallmanifest.Parser,
	installationValidator biinstallmanifest.Validator,
	deploymentManifestPath string,
	tarballProvider bitarball.Provider,

) DeploymentDeleter {
	return DeploymentDeleter{
		ui:     ui,
		logTag: logTag,
		logger: logger,
		fs:     fs,
		deploymentStateService:   deploymentStateService,
		releaseManager:           releaseManager,
		installerFactory:         installerFactory,
		cloudFactory:             cloudFactory,
		agentClientFactory:       agentClientFactory,
		blobstoreFactory:         blobstoreFactory,
		deploymentManagerFactory: deploymentManagerFactory,
		releaseSetParser:         releaseSetParser,
		releaseSetValidator:      releaseSetValidator,
		releaseExtractor:         releaseExtractor,
		installationParser:       installationParser,
		installationValidator:    installationValidator,
		deploymentManifestPath:   deploymentManifestPath,
		tarballProvider:          tarballProvider,
	}
}

type DeploymentDeleter struct {
	ui                       biui.UI
	logTag                   string
	logger                   boshlog.Logger
	fs                       boshsys.FileSystem
	deploymentStateService   biconfig.DeploymentStateService
	releaseManager           birel.Manager
	installerFactory         biinstall.InstallerFactory
	cloudFactory             bicloud.Factory
	agentClientFactory       bihttpagent.AgentClientFactory
	blobstoreFactory         biblobstore.Factory
	deploymentManagerFactory bidepl.ManagerFactory
	releaseSetParser         birelsetmanifest.Parser
	releaseSetValidator      birelsetmanifest.Validator
	releaseExtractor         birel.Extractor
	installationParser       biinstallmanifest.Parser
	installationValidator    biinstallmanifest.Validator
	deploymentManifestPath   string
	tarballProvider          bitarball.Provider
}

func (c *DeploymentDeleter) DeleteDeployment(stage biui.Stage) (err error) {
	c.ui.PrintLinef("Deployment state: '%s'", c.deploymentStateService.Path())

	if !c.deploymentStateService.Exists() {
		c.ui.PrintLinef("No deployment state file found.")
		return nil
	}

	deploymentState, err := c.deploymentStateService.Load()
	if err != nil {
		return bosherr.WrapError(err, "Loading deployment state")
	}

	defer func() {
		err := c.releaseManager.DeleteAll()
		if err != nil {
			c.logger.Warn(c.logTag, "Deleting all extracted releases: %s", err.Error())
		}
	}()

	var installationManifest biinstallmanifest.Manifest
	err = stage.PerformComplex("validating", func(stage biui.Stage) error {
		var err error
		var cpiReleaseRef birelmanifest.ReleaseRef

		installationManifest, cpiReleaseRef, err = c.parseDeploymentManifest(stage, c.deploymentManifestPath)
		if err != nil {
			return err
		}

		releasePath, err := c.tarballProvider.Get(bitarball.Source(cpiReleaseRef), stage)
		if err != nil {
			return err
		}

		err = stage.Perform(fmt.Sprintf("Validating release '%s'", cpiReleaseRef.Name), func() error {
			cpiRelease, err := c.releaseExtractor.Extract(releasePath)
			if err != nil {
				return bosherr.WrapErrorf(err, "Extracting release '%s'", releasePath)
			}
			c.releaseManager.Add(cpiRelease)

			cpiReleaseRefJobName := installationManifest.Template.Name
			err = bicpirel.NewValidator().Validate(cpiRelease, cpiReleaseRefJobName)
			if err != nil {
				return bosherr.WrapErrorf(err, "Invalid CPI release '%s'", cpiRelease.Name())
			}

			return nil
		})

		return err
	})
	if err != nil {
		return err
	}

	installer, err := c.installerFactory.NewInstaller()
	if err != nil {
		return bosherr.WrapError(err, "Creating CPI Installer")
	}

	var installation biinstall.Installation
	err = stage.PerformComplex("installing CPI", func(installStage biui.Stage) error {
		installation, err = installer.Install(installationManifest, installStage)
		return err
	})
	if err != nil {
		return err
	}

	err = stage.Perform("Starting registry", func() error {
		return installation.StartRegistry()
	})
	if err != nil {
		return err
	}
	defer func() {
		//TODO: wrap stopping registry in stage?
		err := installation.StopRegistry()
		if err != nil {
			c.logger.Warn(c.logTag, "Registry failed to stop: %s", err)
		}
	}()

	c.logger.Debug(c.logTag, "Creating cloud client...")
	cloud, err := c.cloudFactory.NewCloud(installation, deploymentState.DirectorID)
	if err != nil {
		return bosherr.WrapError(err, "Creating CPI client from CPI installation")
	}

	c.logger.Debug(c.logTag, "Creating agent client...")
	agentClient := c.agentClientFactory.NewAgentClient(deploymentState.DirectorID, installationManifest.Mbus)

	c.logger.Debug(c.logTag, "Creating blobstore client...")
	blobstore, err := c.blobstoreFactory.Create(installationManifest.Mbus)
	if err != nil {
		return bosherr.WrapError(err, "Creating blobstore client")
	}

	c.logger.Debug(c.logTag, "Creating deployment manager...")
	deploymentManager := c.deploymentManagerFactory.NewManager(cloud, agentClient, blobstore)

	c.logger.Debug(c.logTag, "Finding current deployment...")
	deployment, found, err := deploymentManager.FindCurrent()
	if err != nil {
		return bosherr.WrapError(err, "Finding current deployment")
	}

	err = stage.PerformComplex("deleting deployment", func(deleteStage biui.Stage) error {
		if !found {
			//TODO: skip? would require adding skip support to PerformComplex
			c.logger.Debug(c.logTag, "No current deployment found...")
			return nil
		}

		return deployment.Delete(deleteStage)
	})
	if err != nil {
		return bosherr.WrapError(err, "Deleting deployment")
	}

	return deploymentManager.Cleanup(stage)
}

func (c *DeploymentDeleter) parseDeploymentManifest(validationStage biui.Stage, deploymentManifestPath string) (
	biinstallmanifest.Manifest,
	birelmanifest.ReleaseRef,
	error,
) {
	var cpiRelease birelmanifest.ReleaseRef
	var releaseSetManifest birelsetmanifest.Manifest
	var installationManifest biinstallmanifest.Manifest

	err := validationStage.Perform("Validating deployment manifest", func() error {
		var err error
		installationManifest, err = c.installationParser.Parse(deploymentManifestPath)
		if err != nil {
			return bosherr.WrapErrorf(err, "Parsing installation manifest '%s'", deploymentManifestPath)
		}

		releaseSetManifest, err = c.releaseSetParser.Parse(deploymentManifestPath)
		if err != nil {
			return bosherr.WrapErrorf(err, "Parsing release set manifest '%s'", deploymentManifestPath)
		}

		err = c.releaseSetValidator.Validate(releaseSetManifest)
		if err != nil {
			return bosherr.WrapError(err, "Validating release set manifest")
		}

		err = c.installationValidator.Validate(installationManifest, releaseSetManifest)
		if err != nil {
			return bosherr.WrapError(err, "Validating installation manifest")
		}
		cpiReleaseName := installationManifest.Template.Release

		var found bool
		cpiRelease, found = releaseSetManifest.FindByName(cpiReleaseName)
		if !found {
			return bosherr.Errorf("installation release '%s' must refer to a release in releases", cpiReleaseName)
		}

		return nil
	})
	return installationManifest, cpiRelease, err
}
