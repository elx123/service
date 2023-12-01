// SocketConfig is configuration relevant to the transport socket and protocol.
package config

import "time"

type SocketConfig struct {
	PongWaitMs           int   `default:"25000"`
	PingPeriodMs         int   `default:"15000"`
	WriteWaitMs          int   `default:"5000"`
	PingBackoffThreshold int   `default:"20"`
	OutgoingQueueSize    int   `default:"64"`
	MaxMessageSizeBytes  int64 `default:"4096"`
	ReadBufferSizeBytes  int   `default:"4096"`
	WriteBufferSizeBytes int   `default:"4096"`
}

// SessionConfig is configuration relevant to the session.
type SessionConfig struct {
	SingleSocket bool `default:"true"`
}

type Web struct {
	APIHost         string        `default:"0.0.0.0:3000"`
	DebugHost       string        `default:"0.0.0.0:4000"`
	ReadTimeout     time.Duration `default:"5s"`
	WriteTimeout    time.Duration `default:"10s"`
	IdleTimeout     time.Duration `default:"120s"`
	ShutdownTimeout time.Duration `default:"20s"`
}

type Auth struct {
	KeysFolder string `default:"zarf/keys/"`
	ActiveKID  string `default:"54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"`
}

type DB struct {
	User         string `default:"postgres"`
	Password     string `default:"postgres,mask"`
	Host         string `default:"localhost"`
	Name         string `default:"postgres"`
	MaxIdleConns int    `default:"0"`
	MaxOpenConns int    `default:"0"`
	DisableTLS   bool   `default:"true"`
}

type Zipkin struct {
	ReporterURI string  `default:"http://localhost:9411/api/v2/spans"`
	ServiceName string  `default:"sales-api"`
	Probability float64 `default:"0.05"`
}
