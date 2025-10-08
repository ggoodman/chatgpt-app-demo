package main

type Config struct {
	Port          int    `env:"PORT,default=8080"`
	PublicUrl     string `env:"PUBLIC_URL,default=http://localhost:8080"`
	RedisUrl      string `env:"REDIS_URL,default=redis://localhost:6379"`
	AuthIssuerUrl string `env:"AUTH_ISSUER_URL,default=http://localhost:8081"`
}
