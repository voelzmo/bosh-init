package devicepathresolver

import (
	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type virtioDevicePathResolver struct {
	idDevicePathResolver     DevicePathResolver
	mappedDevicePathResolver DevicePathResolver
	logger                   boshlog.Logger
	logTag                   string
}

func NewVirtioDevicePathResolver(
	idDevicePathResolver DevicePathResolver,
	mappedDevicePathResolver DevicePathResolver,
	logger boshlog.Logger,
) DevicePathResolver {
	return virtioDevicePathResolver{
		idDevicePathResolver:     idDevicePathResolver,
		mappedDevicePathResolver: mappedDevicePathResolver,
		logger: logger,
		logTag: "virtioDevicePathResolver",
	}
}

func (vpr virtioDevicePathResolver) GetRealDevicePath(diskSettings boshsettings.DiskSettings) (string, bool, error) {
	realPath, timeout, err := vpr.idDevicePathResolver.GetRealDevicePath(diskSettings)
	if err == nil {
		vpr.logger.Debug(vpr.logTag, "Resolved disk '%s' by ID '%s' as '%s'", diskSettings.Path, diskSettings.ID, realPath)
		return realPath, false, nil
	}

	vpr.logger.Debug(vpr.logTag,
		"Failed to get device real path by disk ID: '%s'. Error: '%s', timeout: '%t'",
		diskSettings.ID,
		err.Error(),
		timeout,
	)

	vpr.logger.Debug(vpr.logTag, "Using mapped resolver to get device real path")

	realPath, timeout, err = vpr.mappedDevicePathResolver.GetRealDevicePath(diskSettings)
	if err != nil {
		return "", timeout, bosherr.WrapError(err, "Resolving mapped device path")
	}

	return realPath, false, nil
}
