package config

type TiDBConfigInfo map[string]*TiDBConfig

type TiKVConfigInfo map[string]*TiKVConfig

type PDConfigInfo map[string]*PDConfig

type TiDBConfig struct {
	TokenLimit      int `toml:"token-limit"`
	TxnLocalLatches struct {
		Enabled bool `toml:"enabled"`
	} `toml:"txn-local-latches"`
}

type TiKVConfig struct {
	Server struct {
		GrpcConcurrency int `toml:"grpc-concurrency"`
	} `toml:"server"`
	Storage struct {
		SchedulerWorkerPoolSize int `toml:"scheduler-worker-pool-size"`
	} `toml:"storage"`
	RaftStore struct {
		StorePoolSize int `toml:"store-pool-size"`
		ApplyPoolSize int `toml:"apply-pool-size"`
	} `toml:"raftstore"`
	ReadPool struct {
		Storage struct {
			HighConcurrency   int `toml:"high-concurrency"`
			NormalConcurrency int `toml:"normal-concurrency"`
			LowConcurrency    int `toml:"low-concurrency"`
		} `toml:"storage"`
		Coprocessor struct {
			HighConcurrency   int `toml:"high-concurrency"`
			NormalConcurrency int `toml:"normal-concurrency"`
			LowConcurrency    int `toml:"low-concurrency"`
		} `toml:"coprocessor"`
	} `toml:"readpool"`
}

type PDConfig struct {
	Schedule struct {
		LeaderScheduleLimit int `toml:"leader-schedule-limit"`
	} `schedule`
}
