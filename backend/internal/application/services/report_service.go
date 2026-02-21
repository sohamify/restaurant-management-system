package services

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrInvalidReportPeriod = errors.New("invalid report period")
	ErrDatesRequired       = errors.New("start_date and end_date are required")
)

type ReportService interface {
	GetSalesSummary(ctx context.Context, filter entities.ReportFilter) (*entities.SalesSummary, error)
	GetTopSellingItems(ctx context.Context, filter entities.ReportFilter, limit int) ([]*entities.TopItem, error)
	GetInventoryStatus(ctx context.Context) ([]*entities.InventoryStatusItem, error)
	GetPaymentMethodBreakdown(ctx context.Context, filter entities.ReportFilter) ([]*entities.PaymentBreakdown, error)
	GetRefundSummary(ctx context.Context, filter entities.ReportFilter) (*entities.RefundSummary, error)
	GetRevenueForecast(ctx context.Context) (*entities.RevenueForecast, error)
}

type reportService struct {
	reportRepo repositories.ReportRepository
	logger     *zap.Logger
}

func NewReportService(
	reportRepo repositories.ReportRepository,
	logger *zap.Logger,
) ReportService {
	return &reportService{
		reportRepo: reportRepo,
		logger:     logger,
	}
}

func (s *reportService) GetSalesSummary(ctx context.Context, filter entities.ReportFilter) (*entities.SalesSummary, error) {
	if filter.StartDate.IsZero() || filter.EndDate.IsZero() {
		return nil, ErrDatesRequired
	}

	summary, err := s.reportRepo.AggregateSales(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get sales summary", zap.Error(err))
		return nil, err
	}

	return summary, nil
}

func (s *reportService) GetTopSellingItems(ctx context.Context, filter entities.ReportFilter, limit int) ([]*entities.TopItem, error) {
	if filter.StartDate.IsZero() || filter.EndDate.IsZero() {
		return nil, ErrDatesRequired
	}

	if limit <= 0 {
		limit = 10
	}

	items, err := s.reportRepo.AggregateTopItems(ctx, filter, limit)
	if err != nil {
		s.logger.Error("Failed to get top selling items", zap.Error(err))
		return nil, err
	}

	return items, nil
}

func (s *reportService) GetInventoryStatus(ctx context.Context) ([]*entities.InventoryStatusItem, error) {
	status, err := s.reportRepo.GetInventoryStatus(ctx)
	if err != nil {
		s.logger.Error("Failed to get inventory status", zap.Error(err))
		return nil, err
	}

	return status.Items, nil
}

func (s *reportService) GetPaymentMethodBreakdown(ctx context.Context, filter entities.ReportFilter) ([]*entities.PaymentBreakdown, error) {
	if filter.StartDate.IsZero() || filter.EndDate.IsZero() {
		return nil, ErrDatesRequired
	}

	breakdown, err := s.reportRepo.AggregatePaymentBreakdown(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get payment breakdown", zap.Error(err))
		return nil, err
	}

	return breakdown, nil
}

func (s *reportService) GetRefundSummary(ctx context.Context, filter entities.ReportFilter) (*entities.RefundSummary, error) {
	if filter.StartDate.IsZero() || filter.EndDate.IsZero() {
		return nil, ErrDatesRequired
	}

	summary, err := s.reportRepo.AggregateRefundSummary(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get refund summary", zap.Error(err))
		return nil, err
	}

	return summary, nil
}

func (s *reportService) GetRevenueForecast(ctx context.Context) (*entities.RevenueForecast, error) {
	// Requesting last 7 days of revenue from repository
	pastData, err := s.reportRepo.GetPastRevenueData(ctx, 7)
	if err != nil {
		s.logger.Error("Failed to get past data for forecast", zap.Error(err))
		return nil, err
	}

	if len(pastData) == 0 {
		return &entities.RevenueForecast{NextPeriodRevenue: 0}, nil
	}

	// Simple Moving Average (SMA) Logic
	var sum float64
	for _, v := range pastData {
		sum += v
	}

	forecast := sum / float64(len(pastData))

	return &entities.RevenueForecast{
		NextPeriodRevenue: forecast,
	}, nil
}
