package machineset

import (
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	. "github.com/onsi/gomega"
	"testing"
	"time"
)

type MockCacheOpenStackInstanceService struct {
	flavors             []*flavors.Flavor
	GetFlavorIDCalled   int
	GetFlavorInfoCalled int
}

func (mock *MockCacheOpenStackInstanceService) GetFlavorID(flavorName string) (string, error) {
	mock.GetFlavorIDCalled += 1
	for _, flavor := range mock.flavors {
		if flavor.Name == flavorName {
			return flavor.ID, nil
		}
	}
	return "", fmt.Errorf("could not find flavor id for %v", flavorName)
}

func (mock *MockCacheOpenStackInstanceService) GetFlavorInfo(flavorID string) (flavor *flavors.Flavor, err error) {
	mock.GetFlavorInfoCalled += 1
	for _, flavor := range mock.flavors {
		if flavor.ID == flavorID {
			return flavor, nil
		}
	}
	return nil, fmt.Errorf("could not find flavor with id %v", flavorID)
}

func (mock *MockCacheOpenStackInstanceService) ResetCallCounts() {
	mock.GetFlavorIDCalled = 0
	mock.GetFlavorInfoCalled = 0
}

var knownNotInCacheFlavor = &flavors.Flavor{
	ID:   "knownNotInCacheFlavorId",
	Name: "knownNotInCacheFlavor",
}
var knownInCacheFlavor = &flavors.Flavor{
	ID:   "knownInCacheFlavorId",
	Name: "knownInCacheFlavor",
}
var staledTimeExceededFlavor = &flavors.Flavor{
	ID:   "staledTimeExceededFlavorId",
	Name: "staledTimeExceededFlavor",
}
var failedTimeFlavor = &flavors.Flavor{
	ID:   "failedTimeFlavorId",
	Name: "failedTimeFlavor",
}

var mockOSInstance = MockCacheOpenStackInstanceService{
	GetFlavorIDCalled:   0,
	GetFlavorInfoCalled: 0,
	flavors: []*flavors.Flavor{
		knownNotInCacheFlavor,
		knownInCacheFlavor,
		staledTimeExceededFlavor,
		failedTimeFlavor,
	},
}

func Test_machineFlavorsCache_getFlavorInfo(t *testing.T) {
	testCases := []struct {
		name                         string
		flavorName                   string
		flavorInfoExpected           *flavors.Flavor
		prepCache                    []string
		StaledTime                   time.Duration
		RefreshFailureTime           time.Duration
		SleepTime                    time.Duration
		ExpectedCallsToGetFlavorInfo int
		ExpectedCallsToGetFlavorID   int
	}{
		{
			name:                         "valid flavor not in cache",
			flavorName:                   knownNotInCacheFlavor.Name,
			flavorInfoExpected:           knownNotInCacheFlavor,
			ExpectedCallsToGetFlavorInfo: 1,
			ExpectedCallsToGetFlavorID:   1,
			StaledTime:                   300 * time.Second,
			RefreshFailureTime:           60 * time.Second,
			SleepTime:                    1 * time.Second,
		},
		{
			name:                         "valid flavor in cache",
			flavorName:                   knownInCacheFlavor.Name,
			flavorInfoExpected:           knownInCacheFlavor,
			prepCache:                    []string{knownInCacheFlavor.Name},
			ExpectedCallsToGetFlavorInfo: 0,
			ExpectedCallsToGetFlavorID:   0,
			StaledTime:                   300 * time.Second,
			RefreshFailureTime:           60 * time.Second,
			SleepTime:                    1 * time.Second,
		},
		{
			name:                         "unknown flavor is flagged in cache.",
			flavorName:                   "invalidFlavorName",
			flavorInfoExpected:           nil,
			ExpectedCallsToGetFlavorInfo: 0,
			ExpectedCallsToGetFlavorID:   0,
			StaledTime:                   300 * time.Second,
			RefreshFailureTime:           60 * time.Second,
			SleepTime:                    1 * time.Second,
		},
		{
			name:                         "staled time is exceeded",
			flavorName:                   staledTimeExceededFlavor.Name,
			flavorInfoExpected:           staledTimeExceededFlavor,
			ExpectedCallsToGetFlavorInfo: 1,
			ExpectedCallsToGetFlavorID:   1,
			StaledTime:                   2 * time.Second,
			RefreshFailureTime:           1 * time.Second,
			SleepTime:                    3 * time.Second,
		},
		{
			name:                         "failed time is respected",
			flavorName:                   "invalidFlavorName",
			flavorInfoExpected:           nil,
			prepCache:                    []string{"invalidFlavorName"},
			ExpectedCallsToGetFlavorInfo: 0,
			ExpectedCallsToGetFlavorID:   0,
			StaledTime:                   300 * time.Second,
			RefreshFailureTime:           60 * time.Second,
			SleepTime:                    1 * time.Second,
		},
		{
			name:                         "failed time is exceeded",
			flavorName:                   "invalidFlavorName",
			flavorInfoExpected:           nil,
			prepCache:                    []string{"invalidFlavorName"},
			ExpectedCallsToGetFlavorInfo: 0,
			ExpectedCallsToGetFlavorID:   1,
			StaledTime:                   300 * time.Second,
			RefreshFailureTime:           1 * time.Second,
			SleepTime:                    5 * time.Second,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			g := NewWithT(tt)
			mfc := newMachineFlavorCache()
			g.Expect(mfc).ShouldNot(BeNil())
			for _, prepFlavor := range tc.prepCache {
				_ = mfc.getFlavorInfo(&mockOSInstance, prepFlavor)
			}
			g.Expect(mfc.cache).To(HaveLen(len(tc.prepCache)))

			mockOSInstance.ResetCallCounts()
			mfc.staledTime = tc.StaledTime
			mfc.refreshFailureTime = tc.RefreshFailureTime
			time.Sleep(tc.SleepTime)
			flavorInfo := mfc.getFlavorInfo(&mockOSInstance, tc.flavorName)
			g.Expect(flavorInfo).To(Equal(tc.flavorInfoExpected))
			g.Expect(mockOSInstance.GetFlavorInfoCalled).To(Equal(tc.ExpectedCallsToGetFlavorInfo))
			//g.Expect(mockOSInstance.GetFlavorIDCalled != tc.ExpectedCallsToGetFlavorID)
		})

	}
}
