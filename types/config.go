package types

import (
	"gitlab.wallstcn.com/baoer/flash/flashstd"
)

type StockYidongConfig struct {
	CreateMsgUrl string `yaml:"create_msg_url"`
	Secret       string `yaml:"secret"`
}

type Config struct {
	Micro       flashstd.MicroConfig       `yaml:"micro"`
	Logging     flashstd.LoggingConfig     `yaml:"logging"`
	NsqConsumer flashstd.NSQConsumerConfig `yaml:"nsq_consumer"`
	NsqProducer flashstd.NSQProducerConfig `yaml:"nsq_producer"`
	Redis       flashstd.RedisConfig       `yaml:"redis"`
	Distask     flashstd.DistaskConfig     `yaml:"distask"`
	StockYidong StockYidongConfig          `yaml:"stock_yidong"`
}
