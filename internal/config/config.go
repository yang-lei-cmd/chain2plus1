package config

import (
	"os"
)

// Config 全局配置 (Phase 5: 新增第三方支付配置)
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Log      LogConfig
	Payment  PaymentConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port       string `mapstructure:"PORT"`
	Mode       string `mapstructure:"MODE"`
	Host       string `mapstructure:"HOST"`
	ReadTimeout int  `mapstructure:"READ_TIMEOUT"`
	WriteTimeout int `mapstructure:"WRITE_TIMEOUT"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
	Charset  string `mapstructure:"CHARSET"`
	Loc      string `mapstructure:"LOC"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string `mapstructure:"JWT_SECRET"`
	ExpireDays int    `mapstructure:"JWT_EXPIRE_DAYS"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"LOG_LEVEL"`
	Output string `mapstructure:"LOG_OUTPUT"` // console or file
}

// LoadConfig 加载配置 (Phase 5: 新增支付配置加载)
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:       getEnv("PORT", "8080"),
			Mode:       getEnv("MODE", "debug"),
			Host:       getEnv("HOST", "0.0.0.0"),
			ReadTimeout:  5,
			WriteTimeout: 10,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", "Linqi@2024"),
			Name:     getEnv("DB_NAME", "chain2plus1"),
			Charset:  getEnv("CHARSET", "utf8mb4"),
			Loc:      getEnv("LOC", "Local"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "Linqi@2024"),
			ExpireDays: 7,
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Output: getEnv("LOG_OUTPUT", "console"),
		},
		Payment: PaymentConfig{
			DefaultFeeRate:  getFloatEnv("DEFAULT_FEE_RATE", 0.006), // 0.6%
			PaymentTimeout:  getIntEnv("PAYMENT_TIMEOUT_MINUTES", 30), // 30分钟
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		var result int
		// Simple parsing
		for _, c := range value {
			if c >= '0' && c <= '9' {
				result = result*10 + int(c-'0')
			}
		}
		if result > 0 {
			return result
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		var result float64
		hasDot := false
		divisor := 1.0
		for _, c := range value {
			if c >= '0' && c <= '9' {
				result = result*10 + float64(c-'0')
			} else if c == '.' {
				hasDot = true
			} else if hasDot {
				divisor *= 10
				result += float64(c-'0') / divisor
			}
		}
		if hasDot || result > 0 {
			return result
		}
	}
	return defaultValue
}
