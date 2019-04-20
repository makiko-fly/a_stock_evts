package main

import (
	// "github.com/micro/go-micro/client"
	"github.com/jinzhu/configor"
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
	"gitlab.wallstcn.com/baoer/flash/flashstd"
	"gitlab.wallstcn.com/baoer/flash/flashstd/config"
	"gitlab.wallstcn.com/baoer/flash/flashtick/business"
	"gitlab.wallstcn.com/baoer/flash/flashtick/dao"
	"gitlab.wallstcn.com/baoer/flash/flashtick/g"
	"gitlab.wallstcn.com/baoer/flash/flashtick/redislogger"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
)

func main() {
	conf := new(types.Config)
	config.LoadConfig(conf, "conf/flashtick.yaml")

	g.SysConf = conf
	if err := conf.Logging.Init(); err != nil {
		panic(err)
	}
	flashstd.Infoln("Logging initialized.")

	flashstd.Infoln("Start redis client...")
	redisClient := conf.Redis.NewRedisClient()
	if err := redisClient.Ping().Err(); err != nil {
		flashstd.Panicf("Failed to ping redis, err: %v", err)
	}
	redislogger.InitWithRedisClient(redisClient)
	redislogger.Printf("RedisLogger initialized")
	g.MinutePriceCache = dao.NewMinPriceCacheDAO(redisClient)
	g.StockStatCache = dao.NewStockStatCacheDAO(redisClient)
	g.PremarketCache = dao.NewPremarketCacheDAO(redisClient)
	g.StockEvtCache = dao.NewStockEvtCacheDAO(redisClient)
	g.FreshStockCache = dao.NewFreshStockCacheDAO(redisClient)
	g.D5AvgVolCache = dao.NewD5AvgVolCacheDAO(redisClient)

	service := flashstd.NewMicroService(conf.Micro)
	g.ExternalClient = flashcommon.GetExternalClient(service.Client())

	flashstd.Infoln("Start stock feeds producer...")
	nsqProducer := conf.NsqProducer.NewNSQProducer()
	if err := nsqProducer.Ping(); err != nil {
		flashstd.Panicf("Failed to ping nsq producer, err: %v", err)
	} else {
		g.NsqProducer = nsqProducer
	}

	flashstd.Infoln("Start distributed task worker...")
	g.TaskWorker = conf.Distask.NewWorker()
	taskName := "flashtick.TaskWorker.DailyInit"
	if err := g.TaskWorker.Register(taskName, business.DailyInit); err != nil {
		flashstd.Panicf("Failed to register task %s, err: %v", taskName, err)
	}
	taskName2 := "flashtick.TaskWorker.DailyCleanup"
	if err := g.TaskWorker.Register(taskName2, business.DailyCleanup); err != nil {
		flashstd.Panicf("Failed to register task %s, err: %v", taskName2, err)
	}

	if configor.ENV() == "local" { // if we didn't specify CONFIGOR_ENV, it defaults to "local"
		LocalTest()
	}

	// business.DailyInit()

	business.PopulateD5AvgVolMap()
	business.PopulateFreshStocksMap()
	business.PopulateTradingStartMap()

	flashstd.Infoln("Start tick data consumer...")
	nsqConsumer := conf.NsqConsumer.NewNSQConsumer()
	nsqConsumer.AddHandler(business.NewTickDispather(g.ConsumerNum, g.ConsumerBufferSize))
	errChan := make(chan error, 1)
	go nsqConsumer.ServeForever(errChan)
	go g.TaskWorker.ServeForever(errChan)
	flashstd.WaitError(errChan)
}

func LocalTest() {
	// d5AvgVolMap := map[string]int64{}
	// d5AvgVolMap["000666.SS"] = 12345
	// d5AvgVolMap["000777.SZ"] = 54321
	// g.D5AvgVolCache.Set(d5AvgVolMap)
}
