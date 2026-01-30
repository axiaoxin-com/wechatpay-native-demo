// service/order_store.go
package service

import (
	"sync"
	"time"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusNotPay    OrderStatus = "NOTPAY"    // 未支付
	OrderStatusSuccess   OrderStatus = "SUCCESS"   // 支付成功
	OrderStatusClosed    OrderStatus = "CLOSED"    // 已关闭
	OrderStatusRefunding OrderStatus = "REFUNDING" // 退款中
	OrderStatusRefunded  OrderStatus = "REFUNDED"  // 已退款
)

// Order 订单信息
type Order struct {
	ID            string      `json:"id"`
	OutTradeNo    string      `json:"out_trade_no"`
	Description   string      `json:"description"`
	Amount        int64       `json:"amount"` // 单位：分
	Status        OrderStatus `json:"status"`
	CreateTime    time.Time   `json:"create_time"`
	PayTime       *time.Time  `json:"pay_time,omitempty"`
	TransactionID string      `json:"transaction_id,omitempty"`
	RefundNo      string      `json:"refund_no,omitempty"`
	RefundAmount  int64       `json:"refund_amount,omitempty"`
	RefundTime    *time.Time  `json:"refund_time,omitempty"`
}

// OrderStore 内存订单存储
type OrderStore struct {
	mu     sync.RWMutex
	orders map[string]*Order // key: out_trade_no
}

// NewOrderStore 创建订单存储
func NewOrderStore() *OrderStore {
	return &OrderStore{
		orders: make(map[string]*Order),
	}
}

// Save 保存订单
func (s *OrderStore) Save(order *Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.OutTradeNo] = order
}

// Get 获取订单
func (s *OrderStore) Get(outTradeNo string) (*Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[outTradeNo]
	return order, ok
}

// GetAll 获取所有订单（按创建时间倒序）
func (s *OrderStore) GetAll() []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Order, 0, len(s.orders))
	for _, order := range s.orders {
		result = append(result, order)
	}

	// 按创建时间倒序排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreateTime.Before(result[j].CreateTime) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// UpdateStatus 更新订单状态
func (s *OrderStore) UpdateStatus(outTradeNo string, status OrderStatus) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[outTradeNo]
	if !ok {
		return false
	}

	order.Status = status
	now := time.Now()

	switch status {
	case OrderStatusSuccess:
		order.PayTime = &now
	case OrderStatusRefunded:
		order.RefundTime = &now
	}

	return true
}

// UpdatePayInfo 更新支付信息
func (s *OrderStore) UpdatePayInfo(outTradeNo, transactionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[outTradeNo]
	if !ok {
		return false
	}

	order.TransactionID = transactionID
	order.Status = OrderStatusSuccess
	now := time.Now()
	order.PayTime = &now

	return true
}

// UpdateRefundInfo 更新退款信息
func (s *OrderStore) UpdateRefundInfo(outTradeNo, refundNo string, refundAmount int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[outTradeNo]
	if !ok {
		return false
	}

	order.RefundNo = refundNo
	order.RefundAmount = refundAmount
	order.Status = OrderStatusRefunding

	return true
}

// Delete 删除订单（清理功能）
func (s *OrderStore) Delete(outTradeNo string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.orders, outTradeNo)
}
