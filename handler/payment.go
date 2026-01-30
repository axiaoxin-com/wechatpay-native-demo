// handler/payment.go
package handler

import (
	"fmt"
	"net/http"
	"time"

	"wechatpay-native-demo/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	wechatPay  *service.WechatPayService
	orderStore *service.OrderStore
}

func NewPaymentHandler(wechatPay *service.WechatPayService, orderStore *service.OrderStore) *PaymentHandler {
	return &PaymentHandler{
		wechatPay:  wechatPay,
		orderStore: orderStore,
	}
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderID     string `json:"order_id"`
	CodeURL     string `json:"code_url"`
	Amount      int64  `json:"amount"`
	ProductName string `json:"product_name"`
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	ProductName string `json:"product_name" binding:"required"`
	Amount      int64  `json:"amount" binding:"required,min=1"` // 单位：分
	OutTradeNo  string `json:"out_trade_no,omitempty"`          // 可选：已有订单号（重新支付）
}

// CreateOrder 创建支付订单
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var outTradeNo string
	var description string
	var amount int64

	// 如果传入了已有订单号，查询并使用原订单信息
	if req.OutTradeNo != "" {
		existingOrder, ok := h.orderStore.Get(req.OutTradeNo)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
			return
		}

		// 检查订单状态
		if existingOrder.Status != service.OrderStatusNotPay {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "订单状态不允许重新支付",
				"status": existingOrder.Status,
			})
			return
		}

		outTradeNo = req.OutTradeNo
		description = existingOrder.Description
		amount = existingOrder.Amount
	} else {
		// 创建新订单
		outTradeNo = fmt.Sprintf("N%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8])
		description = req.ProductName
		amount = req.Amount

		// 保存新订单到内存
		order := &service.Order{
			ID:          uuid.New().String(),
			OutTradeNo:  outTradeNo,
			Description: description,
			Amount:      amount,
			Status:      service.OrderStatusNotPay,
			CreateTime:  time.Now(),
		}
		h.orderStore.Save(order)
	}

	// 调用微信支付接口创建订单（或重新下单）
	codeURL, err := h.wechatPay.CreateNativeOrder(outTradeNo, description, amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, CreateOrderResponse{
		OrderID:     outTradeNo,
		CodeURL:     codeURL,
		Amount:      amount,
		ProductName: description,
	})
}

// QueryOrder 查询订单状态
func (h *PaymentHandler) QueryOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	// TODO: 从数据库查询订单状态
	order, ok := h.orderStore.Get(orderID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	// 如果订单未支付或支付中，实时查询微信支付状态
	if order.Status == service.OrderStatusNotPay {
		// 调用微信支付查询接口
		resp, err := h.wechatPay.QueryOrder(orderID)
		if err == nil && resp.TradeState != nil {
			// 更新本地状态
			switch *resp.TradeState {
			case "SUCCESS":
				h.orderStore.UpdatePayInfo(orderID, *resp.TransactionId)
			case "CLOSED":
				h.orderStore.UpdateStatus(orderID, service.OrderStatusClosed)
			}
			// 重新获取更新后的订单
			order, _ = h.orderStore.Get(orderID)
		}
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders 获取订单列表
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	orders := h.orderStore.GetAll()
	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  len(orders),
	})
}

// CloseOrder 关闭未支付订单
func (h *PaymentHandler) CloseOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	// 查询订单
	order, ok := h.orderStore.Get(orderID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	// 检查订单状态
	if order.Status != service.OrderStatusNotPay {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "订单状态不允许关闭",
			"status": order.Status,
		})
		return
	}

	// 调用微信支付关闭订单
	err := h.wechatPay.CloseOrder(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新本地状态
	h.orderStore.UpdateStatus(orderID, service.OrderStatusClosed)

	c.JSON(http.StatusOK, gin.H{
		"message":  "订单已关闭",
		"order_id": orderID,
	})
}

// RefundRequest 退款请求
type RefundRequest struct {
	RefundFee int64  `json:"refund_fee"` // 退款金额，单位分，默认全额
	Reason    string `json:"reason"`     // 退款原因
}

// RefundOrder 申请退款
func (h *PaymentHandler) RefundOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有传body，使用默认值
		req.RefundFee = 0
		req.Reason = "用户申请退款"
	}

	// 查询订单
	order, ok := h.orderStore.Get(orderID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	// 检查订单状态
	if order.Status != service.OrderStatusSuccess {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "订单状态不允许退款",
			"status": order.Status,
		})
		return
	}

	// 确定退款金额
	refundFee := req.RefundFee
	if refundFee == 0 || refundFee > order.Amount {
		refundFee = order.Amount // 默认全额退款
	}

	// 生成退款单号
	outRefundNo := fmt.Sprintf("R%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8])

	// 调用微信支付退款接口
	refundResp, err := h.wechatPay.ApplyRefund(orderID, outRefundNo, refundFee, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新本地状态
	h.orderStore.UpdateRefundInfo(orderID, outRefundNo, refundFee)

	c.JSON(http.StatusOK, gin.H{
		"message":       "退款申请已提交",
		"order_id":      orderID,
		"refund_no":     outRefundNo,
		"refund_fee":    refundFee,
		"refund_status": refundResp.Status,
	})
}
