package fakes

import (
	biagentclient "github.com/cloudfoundry/bosh-init/deployment/agentclient"
	bias "github.com/cloudfoundry/bosh-init/deployment/applyspec"
)

type FakeAgentClient struct {
	PingResponses   []pingResponse
	PingCalledCount int

	StopCalled bool
	stopErr    error

	ApplyApplySpec bias.ApplySpec
	ApplyErr       error

	StartCalled bool
	startErr    error

	MountDiskCID string
	mountDiskErr error

	UnmountDiskCID string
	unmountDiskErr error

	listDiskDisks  []string
	listDiskErr    error
	ListDiskCalled bool

	GetStateCalledTimes int
	getStateOutputs     []getStateOutput

	MigrateDiskCalledTimes int
	migrateDiskErr         error
}

type pingResponse struct {
	response string
	err      error
}

type getStateOutput struct {
	state biagentclient.AgentState
	err   error
}

type compilePackageOutput struct {
	blobRef biagentclient.BlobRef
	err     error
}

func NewFakeAgentClient() *FakeAgentClient {
	return &FakeAgentClient{
		getStateOutputs: []getStateOutput{},
	}
}

func (c *FakeAgentClient) Ping() (string, error) {
	c.PingCalledCount++

	if len(c.PingResponses) > 0 {
		response := c.PingResponses[0]
		c.PingResponses = c.PingResponses[1:]
		return response.response, response.err
	}

	return "", nil
}

func (c *FakeAgentClient) Stop() error {
	c.StopCalled = true
	return c.stopErr
}

func (c *FakeAgentClient) Apply(applySpec bias.ApplySpec) error {
	c.ApplyApplySpec = applySpec

	return c.ApplyErr
}

func (c *FakeAgentClient) Start() error {
	c.StartCalled = true
	return c.startErr
}

func (c *FakeAgentClient) GetState() (biagentclient.AgentState, error) {
	c.GetStateCalledTimes++

	getStateReturn := c.getStateOutputs[0]
	c.getStateOutputs = c.getStateOutputs[1:]

	return getStateReturn.state, getStateReturn.err
}

func (c *FakeAgentClient) ListDisk() ([]string, error) {
	c.ListDiskCalled = true
	return c.listDiskDisks, c.listDiskErr
}

func (c *FakeAgentClient) MountDisk(diskCID string) error {
	c.MountDiskCID = diskCID

	return c.mountDiskErr
}

func (c *FakeAgentClient) UnmountDisk(diskCID string) error {
	c.UnmountDiskCID = diskCID
	return c.unmountDiskErr
}

func (c *FakeAgentClient) MigrateDisk() error {
	c.MigrateDiskCalledTimes++
	return c.migrateDiskErr
}

func (c *FakeAgentClient) CompilePackage(
	packageSource biagentclient.BlobRef,
	compiledPackageDependencies []biagentclient.BlobRef,
) (
	compiledPackageRef biagentclient.BlobRef,
	err error,
) {
	return biagentclient.BlobRef{}, nil
}

func (c *FakeAgentClient) SetPingBehavior(response string, err error) {
	c.PingResponses = append(c.PingResponses, pingResponse{
		response: response,
		err:      err,
	})
}

func (c *FakeAgentClient) SetStopBehavior(err error) {
	c.stopErr = err
}

func (c *FakeAgentClient) SetStartBehavior(err error) {
	c.startErr = err
}

func (c *FakeAgentClient) SetGetStateBehavior(stateResponse biagentclient.AgentState, err error) {
	c.getStateOutputs = append(c.getStateOutputs, getStateOutput{
		state: stateResponse,
		err:   err,
	})
}

func (c *FakeAgentClient) SetMountDiskBehavior(err error) {
	c.mountDiskErr = err
}

func (c *FakeAgentClient) SetUnmountDiskBehavior(err error) {
	c.unmountDiskErr = err
}

func (c *FakeAgentClient) SetMigrateDiskBehavior(err error) {
	c.migrateDiskErr = err
}

func (c *FakeAgentClient) SetListDiskBehavior(disks []string, err error) {
	c.listDiskDisks = disks
	c.listDiskErr = err
}
