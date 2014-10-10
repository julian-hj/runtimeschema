package lock_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const HEARTBEAT_INTERVAL = 30 * time.Second

type LockBBS struct {
	store  storeadapter.StoreAdapter
	logger lager.Logger
}

func New(store storeadapter.StoreAdapter, logger lager.Logger) *LockBBS {
	return &LockBBS{
		store:  store,
		logger: logger,
	}
}

func (bbs *LockBBS) NewAuctioneerLock(auctioneerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, shared.LockSchemaPath("auctioneer_lock"), auctioneerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewConvergeLock(convergerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, shared.LockSchemaPath("converge_lock"), convergerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewNsyncBulkerLock(bulkerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, shared.LockSchemaPath("nsync_bulker_lock"), bulkerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewNsyncListenerLock(listenerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, shared.LockSchemaPath("nsync_listener_lock"), listenerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewRouteEmitterLock(emitterID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, shared.LockSchemaPath("route_emitter_lock"), emitterID, interval, bbs.logger)
}
