// config/config.go
package config

import (
	"os"
)

type WechatPayConfig struct {
	AppID          string // 公众账号ID
	MchID          string // 商户号
	APIv3Key       string // APIv3密钥
	CertSerialNo   string // 商户证书序列号
	PrivateKeyPath string // 商户私钥文件路径
	PublicKeyID    string // 微信支付公钥ID
	PublicKeyPath  string // 微信支付公钥文件路径
	NotifyURL      string // 支付结果回调地址
}

func LoadConfig() *WechatPayConfig {
	return &WechatPayConfig{
		AppID:          os.Getenv("WXPAY_APPID"),
		MchID:          os.Getenv("WXPAY_MCHID"),
		APIv3Key:       os.Getenv("WXPAY_APIV3_KEY"),
		CertSerialNo:   os.Getenv("WXPAY_CERT_SERIAL_NO"),
		PrivateKeyPath: os.Getenv("WXPAY_PRIVATE_KEY_PATH"),
		PublicKeyID:    os.Getenv("WXPAY_PUBLIC_KEY_ID"),
		PublicKeyPath:  os.Getenv("WXPAY_PUBLIC_KEY_PATH"),
		NotifyURL:      os.Getenv("WXPAY_NOTIFY_URL"),
	}
}
