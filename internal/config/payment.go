package config

// PaymentConfig 第三方支付配置 (Phase 5)
type PaymentConfig struct {
	WechatAppID      string `mapstructure:"WECHAT_APP_ID"`
	WechatMCHID      string `mapstructure:"WECHAT_MCH_ID"`
	WechatAPIKey     string `mapstructure:"WECHAT_API_KEY"`
	WechatNotifyURL  string `mapstructure:"WECHAT_NOTIFY_URL"`

	AlipayAppID      string `mapstructure:"ALIPAY_APP_ID"`
	AlipayPrivateKey string `mapstructure:"ALIPAY_PRIVATE_KEY"`
	AlipayPublicKey  string `mapstructure:"ALIPAY_PUBLIC_KEY"`
	AlipayNotifyURL  string `mapstructure:"ALIPAY_NOTIFY_URL"`

	DefaultFeeRate   float64 `mapstructure:"DEFAULT_FEE_RATE"` // 支付渠道手续费率
	PaymentTimeout   int     `mapstructure:"PAYMENT_TIMEOUT_MINUTES"` // 支付超时(分钟)
}
