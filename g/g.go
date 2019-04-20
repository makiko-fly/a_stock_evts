package g

import (
	"github.com/nsqio/go-nsq"
	"gitlab.wallstcn.com/baoer/flash/flashcommon/protos/external"
	"gitlab.wallstcn.com/baoer/flash/flashstd"
	"gitlab.wallstcn.com/baoer/flash/flashtick/dao"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
)

// Cache
var StockEvtCache *dao.StockEvtCacheDAO
var MinutePriceCache *dao.MinPriceCacheDAO
var StockStatCache *dao.StockStatCacheDAO
var PremarketCache *dao.PremarketCacheDAO
var FreshStockCache *dao.FreshStockCacheDAO
var D5AvgVolCache *dao.D5AvgVolCacheDAO

// RPC Clients
var ExternalClient external.ExternalServiceClient

// nsq related
var NsqProducer *nsq.Producer
var RealFeedsTopic string = "flash.tick.stock.topic"
var EvtFeedsTopic string = "flash.tick.evts.topic"
var PoolFeedsTopic string = "flash.tick.pool.topic"

// consumer buffer size
var ConsumerNum = 40
var ConsumerBufferSize = 2

// distributed task worker
var TaskWorker *flashstd.DistaskWorker

var SysConf *types.Config
