package entities

import "time"

type SalesSummary struct {
	Period             string  `json:"period"` // e.g. "2025-02-21" for daily
	TotalRevenue       float64 `json:"total_revenue"`
	OrderCount         int     `json:"order_count"`
	AvgOrderValue      float64 `json:"avg_order_value"`
	CompletedOrders    int     `json:"completed_orders"`
	CancelledOrders    int     `json:"cancelled_orders"`
	CancellationRate   float64 `json:"cancellation_rate"`    // percentage
	AvgPreparationTime float64 `json:"avg_preparation_time"` // in minutes (advanced)
	PeakHour           int     `json:"peak_hour"`            // hour with max orders (0-23)
}

type TopItem struct {
	MenuItemID   string  `json:"menu_item_id"`
	Name         string  `json:"name"`
	QuantitySold int     `json:"quantity_sold"`
	Revenue      float64 `json:"revenue"`
}

type InventoryStatus struct {
	Items         []*InventoryStatusItem `json:"items"`
	LowStockCount int                    `json:"low_stock_count"` // advanced
}

type InventoryStatusItem struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Unit              string  `json:"unit"`
	CurrentQuantity   float64 `json:"current_quantity"`
	LowStockThreshold float64 `json:"low_stock_threshold"`
	IsLowStock        bool    `json:"is_low_stock"`
}

type PaymentBreakdown struct {
	Method      string  `json:"method"`
	Count       int     `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

type RefundSummary struct {
	TotalRefunded float64 `json:"total_refunded"`
	RefundCount   int     `json:"refund_count"`
	RefundRate    float64 `json:"refund_rate"` // percentage of sales (advanced)
}

type RevenueForecast struct {
	NextPeriodRevenue float64 `json:"next_period_revenue"` // simple linear forecast (advanced)
}

type ReportFilter struct {
	StartDate time.Time    `json:"start_date"`
	EndDate   time.Time    `json:"end_date"`
	Period    ReportPeriod `json:"period"` // daily, weekly, monthly
	Limit     int          `json:"limit"`  // for top items
}

type ReportPeriod string

const (
	ReportDaily   ReportPeriod = "daily"
	ReportWeekly  ReportPeriod = "weekly"
	ReportMonthly ReportPeriod = "monthly"
)

type PaymentMethodBreakdown struct {
	Method      string  `json:"method"`
	Count       int     `json:"count"`
	TotalAmount float64 `json:"totalAmount"`
}
