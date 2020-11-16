package machineset

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"sync"
	"time"
)

const StaledTime time.Duration = 300 * time.Second
const RefreshFailureTime time.Duration = 60 * time.Second // This controls how often we try to get a look at a failed flavor
// machineFalvorKey is used to identify Machine flavor
type machineFlavorKey struct {
	name string
}

type flavorCacheEntry struct {
	flavorInfoPtr *flavors.Flavor
	updateTime    time.Time
	flavorName    string
}

type machineFlavorsCache struct {
	cacheMutex         sync.Mutex
	cache              map[machineFlavorKey]flavorCacheEntry
	staledTime         time.Duration
	refreshFailureTime time.Duration
}

func newMachineFlavorCache() *machineFlavorsCache {
	return &machineFlavorsCache{
		cache:              map[machineFlavorKey]flavorCacheEntry{},
		staledTime:         StaledTime,
		refreshFailureTime: RefreshFailureTime,
	}
}

func (mfc *machineFlavorsCache) getFlavorInfo(osService OpenStackInstanceService, flavorName string) *flavors.Flavor {
	mfc.cacheMutex.Lock()
	defer mfc.cacheMutex.Unlock()
	if entry, ok := mfc.cache[machineFlavorKey{name: flavorName}]; ok {
		// An entry in the cache has been found but we still need to figure out if it is valid

		if entry.flavorInfoPtr != nil && time.Now().Sub(entry.updateTime) < mfc.staledTime {
			// We have current valid entry
			return entry.flavorInfoPtr
		}
		if entry.flavorInfoPtr == nil && time.Now().Sub(entry.updateTime) < mfc.refreshFailureTime {
			// We have an invalid entry but is too soon to try an refresh, so we return nil
			return nil
		}
	}

	flavorID, err := osService.GetFlavorID(flavorName)
	if err != nil {
		// We failed to find flavor. We populate the cache wit a nil variable and return its entry.
		mfc.cache[machineFlavorKey{name: flavorName}] = flavorCacheEntry{
			flavorInfoPtr: nil,
			updateTime:    time.Now(),
			flavorName:    flavorName,
		}
		return nil
	}

	flavorInfo, err := osService.GetFlavorInfo(flavorID)
	if err != nil {
		if err != nil {
			// We failed to find flavor. We populate the cache wit a nil variable and return its entry.
			mfc.cache[machineFlavorKey{name: flavorName}] = flavorCacheEntry{
				flavorInfoPtr: nil,
				updateTime:    time.Now(),
				flavorName:    flavorName,
			}
			return nil
		}
	}

	mfc.cache[machineFlavorKey{name: flavorName}] = flavorCacheEntry{
		flavorInfoPtr: flavorInfo,
		updateTime:    time.Now(),
		flavorName:    flavorName,
	}
	return flavorInfo
}
