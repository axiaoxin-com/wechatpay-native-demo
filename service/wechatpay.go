// service/wechatpay.go
package service

import (
	"context"
	"fmt"

	"wechatpay-native-demo/config"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type WechatPayService struct {
	client *core.Client
	config *config.WechatPayConfig
}

// NewWechatPayService 创建微信支付服务（使用微信支付公钥模式）
func NewWechatPayService(cfg *config.WechatPayConfig) (*WechatPayService, error) {
	// 加载商户私钥
	privateKey, err := utils.LoadPrivateKeyWithPath(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载商户私钥失败: %v", err)
	}

	// 加载微信支付公钥（用于验签）
	wechatPayPublicKey, err := utils.LoadPublicKeyWithPath(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载微信支付公钥失败: %v", err)
	}

	// 创建商户配置（使用公钥模式）
	opts := []core.ClientOption{
		// 使用商户私钥进行请求签名，使用微信支付公钥验证响应
		option.WithWechatPayPublicKeyAuthCipher(
			cfg.MchID,          // 商户号
			cfg.CertSerialNo,   // 商户证书序列号
			privateKey,         // 商户私钥
			cfg.PublicKeyID,    // 微信支付公钥ID
			wechatPayPublicKey, // 微信支付公钥
		),
	}

	// 创建客户端
	client, err := core.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("创建微信支付客户端失败: %v", err)
	}

	return &WechatPayService{
		client: client,
		config: cfg,
	}, nil
}

// CreateNativeOrder 创建Native支付订单
func (s *WechatPayService) CreateNativeOrder(outTradeNo, description string, totalFee int64) (string, error) {
	svc := native.NativeApiService{Client: s.client}

	resp, result, err := svc.Prepay(context.Background(), native.PrepayRequest{
		Appid:       core.String(s.config.AppID),
		Mchid:       core.String(s.config.MchID),
		Description: core.String(description),
		OutTradeNo:  core.String(outTradeNo),
		NotifyUrl:   core.String(s.config.NotifyURL),
		Amount: &native.Amount{
			Total: core.Int64(totalFee),
		},
	})
	if err != nil {
		return "", fmt.Errorf("创建订单失败: %v, result: %+v", err, result)
	}

	return *resp.CodeUrl, nil
}

// QueryOrder 查询订单
func (s *WechatPayService) QueryOrder(outTradeNo string) (*payments.Transaction, error) {
	svc := native.NativeApiService{Client: s.client}

	resp, result, err := svc.QueryOrderByOutTradeNo(context.Background(), native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(outTradeNo),
		Mchid:      core.String(s.config.MchID),
	})
	if err != nil {
		return nil, fmt.Errorf("查询订单失败: %v, result: %+v", err, result)
	}

	return resp, nil
}

// CloseOrder 关闭订单
func (s *WechatPayService) CloseOrder(outTradeNo string) error {
	svc := native.NativeApiService{Client: s.client}

	result, err := svc.CloseOrder(context.Background(), native.CloseOrderRequest{
		OutTradeNo: core.String(outTradeNo),
		Mchid:      core.String(s.config.MchID),
	})
	if err != nil {
		return fmt.Errorf("关闭订单失败: %v, result: %+v", err, result)
	}

	return nil
}

// ApplyRefund 申请退款
func (s *WechatPayService) ApplyRefund(outTradeNo, outRefundNo string, refundFee int64, reason string) (*refunddomestic.Refund, error) {
	svc := refunddomestic.RefundsApiService{Client: s.client}

	resp, result, err := svc.Create(context.Background(), refunddomestic.CreateRequest{
		OutTradeNo:  core.String(outTradeNo),
		OutRefundNo: core.String(outRefundNo),
		Reason:      core.String(reason),
		NotifyUrl:   core.String(s.config.NotifyURL), // 确保传入回调地址
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(refundFee),
			Total:    core.Int64(refundFee),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("申请退款失败: %v, result: %+v", err, result)
	}

	return resp, nil
}
