package config

type Config struct {
	Socket  *SocketConfig
	Session *SessionConfig
	Web     *Web
	Auth    *Auth
	DB      *DB
	Zipkin  *Zipkin
}

// NewConfig constructs a Config struct which represents server settings, and populates it with default values.
func NewConfig() *Config {
	return &Config{
		Socket:  &SocketConfig{},
		Session: &SessionConfig{},
		Web:     &Web{},
		Auth:    &Auth{},
		DB:      &DB{},
		Zipkin:  &Zipkin{},
	}
}
