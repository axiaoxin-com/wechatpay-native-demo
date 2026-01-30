package handler

import (
	"context"
	"fmt"
	"net/http"

	"wechatpay-native-demo/service"

	"github.com/gin-gonic/gin"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

// NotifyHandler 支付回调处理器
type NotifyHandler struct {
	handler    *notify.Handler
	apiV3Key   string
	orderStore *service.OrderStore
}

// NewNotifyHandler 创建支付回调处理器
func NewNotifyHandler(apiV3Key, wechatPayPublicKeyID, publicKeyPath string, orderStore *service.OrderStore) (*NotifyHandler, error) {
	wechatPayPublicKey, err := utils.LoadPublicKeyWithPath(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载微信支付公钥失败: %v", err)
	}

	handler := notify.NewNotifyHandler(
		apiV3Key,
		verifiers.NewSHA256WithRSAPubkeyVerifier(wechatPayPublicKeyID, *wechatPayPublicKey),
	)

	return &NotifyHandler{
		handler:    handler,
		apiV3Key:   apiV3Key,
		orderStore: orderStore,
	}, nil
}

// PaymentNotify 处理支付和退款回调
func (h *NotifyHandler) PaymentNotify(c *gin.Context) {
	fmt.Printf("[Notify] 收到回调通知\n")
	// 先解析通知获取事件类型
	notifyReq, err := h.handler.ParseNotifyRequest(context.Background(), c.Request, nil)
	if err != nil {
		fmt.Printf("[Notify] 解析通知失败: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"code": "FAIL", "message": "验签未通过"})
		return
	}

	fmt.Printf("[Notify] 收到通知: EventType=%s\n", notifyReq.EventType)

	switch notifyReq.EventType {
	case "TRANSACTION.SUCCESS":
		h.handlePaymentNotify(c, notifyReq)
	case "REFUND.SUCCESS", "REFUND.ABNORMAL", "REFUND.CLOSED":
		h.handleRefundNotify(c, notifyReq)
	default:
		fmt.Printf("[Notify] 未知事件类型: %s\n", notifyReq.EventType)
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "未知事件类型"})
	}
}

// handlePaymentNotify 处理支付通知
func (h *NotifyHandler) handlePaymentNotify(c *gin.Context, notifyReq *notify.Request) {
	transaction := new(Transaction)
	if _, err := h.handler.ParseNotifyRequest(context.Background(), c.Request, transaction); err != nil {
		fmt.Printf("[Notify] 解析支付通知失败: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "解析失败"})
		return
	}

	fmt.Printf("[Notify] 支付成功: OrderID=%s\n", transaction.OutTradeNo)

	// 更新订单状态
	h.orderStore.UpdatePayInfo(transaction.OutTradeNo, transaction.TransactionID)

	go processPaymentSuccess(*transaction)

	c.Status(http.StatusNoContent)
}

// handleRefundNotify 处理退款通知
func (h *NotifyHandler) handleRefundNotify(c *gin.Context, notifyReq *notify.Request) {
	refund := new(Refund)
	if _, err := h.handler.ParseNotifyRequest(context.Background(), c.Request, refund); err != nil {
		fmt.Printf("[Notify] 解析退款通知失败: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "解析失败"})
		return
	}

	fmt.Printf("[Notify] 退款通知: Status=%s, OrderID=%s, RefundID=%s\n",
		refund.RefundStatus, refund.OutTradeNo, refund.OutRefundNo)

	// 更新订单状态
	if refund.RefundStatus == "SUCCESS" {
		h.orderStore.UpdateStatus(refund.OutTradeNo, service.OrderStatusRefunded)
	} else if refund.RefundStatus == "CLOSED" {
		// 退款关闭，恢复订单状态为已支付
		h.orderStore.UpdateStatus(refund.OutTradeNo, service.OrderStatusSuccess)
	}

	c.Status(http.StatusNoContent)
}

// Transaction 支付通知
type Transaction struct {
	AppID         string `json:"appid"`
	MchID         string `json:"mchid"`
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	SuccessTime   string `json:"success_time"`
	Amount        struct {
		Total int `json:"total"`
	} `json:"amount"`
}

// Refund 退款通知
type Refund struct {
	OutTradeNo   string `json:"out_trade_no"`
	OutRefundNo  string `json:"out_refund_no"`
	RefundID     string `json:"refund_id"`
	RefundStatus string `json:"refund_status"`
	SuccessTime  string `json:"success_time"`
	Amount       struct {
		Refund int `json:"refund"`
	} `json:"amount"`
}

func processPaymentSuccess(result Transaction) {
	fmt.Printf("[Process] 支付成功: %s, 金额: %d分\n", result.OutTradeNo, result.Amount.Total)
}
