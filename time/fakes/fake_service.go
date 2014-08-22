package fakes

import (
	"time"
)

type FakeService struct {
	NowTimes []time.Time
}

func (f *FakeService) Now() time.Time {
	time := f.NowTimes[0]
	if len(f.NowTimes) > 0 {
		f.NowTimes = f.NowTimes[1:]
	}
	return time
}