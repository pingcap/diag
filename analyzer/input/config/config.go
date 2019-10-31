package config

type TiDBConfigInfo map[string]*TiDBConfig

type TiKVConfigInfo map[string]*TiKVConfig

type PDConfigInfo map[string]*PDConfig

type TiDBConfig struct {
	TokenLimit      int                   `toml:"token-limit"`
	TxnLocalLatches TxnLocalLatchesConfig `toml:"txn-local-latches"`
}

type TxnLocalLatchesConfig struct {
	Enabled bool `toml:"enabled"`
}

type TiKVConfig struct {
	Server    ServerConfig    `toml:"server"`
	Storage   StorageConfig   `toml:"storage"`
	RaftStore RaftStoreConfig `toml:"raftstore"`
	ReadPool  ReadPoolConfig  `toml:"readpool"`
}

type ServerConfig struct {
	GrpcConcurrency int `toml:"grpc-concurrency"`
}

type StorageConfig struct {
	SchedulerWorkerPoolSize int `toml:"scheduler-worker-pool-size"`
}

type RaftStoreConfig struct {
	StorePoolSize int `toml:"store-pool-size"`
	ApplyPoolSize int `toml:"apply-pool-size"`
}

type ReadPoolConfig struct {
	Storage     ConcurrencyConfig `toml:"storage"`
	Coprocessor ConcurrencyConfig `toml:"coprocessor"`
}

type ConcurrencyConfig struct {
	HighConcurrency   int `toml:"high-concurrency"`
	NormalConcurrency int `toml:"normal-concurrency"`
	LowConcurrency    int `toml:"low-concurrency"`
}

type PDConfig struct{}
