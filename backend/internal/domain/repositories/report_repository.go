package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type ReportRepository interface {
	AggregateSales(ctx context.Context, filter entities.ReportFilter) (*entities.SalesSummary, error)
	AggregateTopItems(ctx context.Context, filter entities.ReportFilter, limit int) ([]*entities.TopItem, error)
	GetInventoryStatus(ctx context.Context) (*entities.InventoryStatus, error)
	AggregatePaymentBreakdown(ctx context.Context, filter entities.ReportFilter) ([]*entities.PaymentBreakdown, error)
	AggregateRefundSummary(ctx context.Context, filter entities.ReportFilter) (*entities.RefundSummary, error)
	GetPastRevenueData(ctx context.Context, periods int) ([]float64, error) // for forecast (advanced)
}
