package disk

import (
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	biconfig "github.com/cloudfoundry/bosh-init/config"
)

type ManagerFactory interface {
	NewManager(bicloud.Cloud) Manager
}

type managerFactory struct {
	diskRepo biconfig.DiskRepo
	logger   boshlog.Logger
}

func NewManagerFactory(
	diskRepo biconfig.DiskRepo,
	logger boshlog.Logger,
) ManagerFactory {
	return &managerFactory{
		diskRepo: diskRepo,
		logger:   logger,
	}
}

func (f *managerFactory) NewManager(cloud bicloud.Cloud) Manager {
	return NewManager(cloud, f.diskRepo, f.logger)
}
