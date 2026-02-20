package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrTableNumberExists  = errors.New("table number already exists")
	ErrTableNotFound      = errors.New("table not found")
	ErrInvalidTableStatus = errors.New("invalid table status")
	ErrCannotDeleteActive = errors.New("cannot delete table with active order or occupied status")
)

type TableService interface {
	CreateTable(ctx context.Context, req *entities.CreateTableRequest) (*entities.TableDTO, error)
	ListTables(ctx context.Context, page, limit int, status string) ([]*entities.TableDTO, int64, error)
	GetTable(ctx context.Context, id string) (*entities.TableDTO, error)
	UpdateTable(ctx context.Context, id string, req *entities.UpdateTableRequest) (*entities.TableDTO, error)
	DeleteTable(ctx context.Context, id string) error
	ListAssignedTables(ctx context.Context, waiterID string) ([]*entities.TableDTO, error)
}

type tableService struct {
	tableRepo repositories.TableRepository
	logger    *zap.Logger
}

func NewTableService(tableRepo repositories.TableRepository, logger *zap.Logger) TableService {
	return &tableService{tableRepo, logger}
}

func (s *tableService) CreateTable(ctx context.Context, req *entities.CreateTableRequest) (*entities.TableDTO, error) {
	if req.TableNumber < 1 {
		return nil, errors.New("table number must be positive")
	}
	if req.Capacity < 1 {
		return nil, errors.New("capacity must be at least 1")
	}

	existing, err := s.tableRepo.FindByTableNumber(ctx, req.TableNumber)
	if err != nil {
		s.logger.Error("Failed to check table number uniqueness", zap.Error(err))
		return nil, errors.New("internal error")
	}
	if existing != nil {
		return nil, ErrTableNumberExists
	}

	table := &entities.Table{
		TableNumber: req.TableNumber,
		Capacity:    req.Capacity,
		Status:      entities.TableAvailable,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	created, err := s.tableRepo.Create(ctx, table)
	if err != nil {
		s.logger.Error("Failed to create table", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Table created", zap.Int("number", req.TableNumber), zap.String("id", created.ID.Hex()))
	return created.ToDTO(), nil
}

func (s *tableService) ListTables(ctx context.Context, page, limit int, status string) ([]*entities.TableDTO, int64, error) {
	filter := bson.M{}

	if status != "" {
		if !isValidStatus(status) {
			return nil, 0, ErrInvalidTableStatus
		}
		filter["status"] = status
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"table_number": 1})

	tables, total, err := s.tableRepo.FindAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list tables", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.TableDTO, len(tables))
	for i, t := range tables {
		dtos[i] = t.ToDTO()
	}

	return dtos, total, nil
}

func (s *tableService) GetTable(ctx context.Context, id string) (*entities.TableDTO, error) {
	tid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrTableNotFound
	}

	table, err := s.tableRepo.FindByID(ctx, tid)
	if err != nil {
		s.logger.Error("Failed to get table", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if table == nil {
		return nil, ErrTableNotFound
	}

	return table.ToDTO(), nil
}

func (s *tableService) UpdateTable(ctx context.Context, id string, req *entities.UpdateTableRequest) (*entities.TableDTO, error) {
	tid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrTableNotFound
	}

	table, err := s.tableRepo.FindByID(ctx, tid)
	if err != nil || table == nil {
		return nil, ErrTableNotFound
	}

	updateFields := bson.M{}

	if req.Capacity != nil {
		if *req.Capacity < 1 {
			return nil, errors.New("capacity must be at least 1")
		}
		updateFields["capacity"] = *req.Capacity
	}

	if req.Status != nil {
		status := entities.TableStatus(*req.Status)
		if !isValidStatus(string(status)) {
			return nil, ErrInvalidTableStatus
		}
		updateFields["status"] = status
	}

	if req.AssignedWaiterID != nil {
		wid, err := primitive.ObjectIDFromHex(*req.AssignedWaiterID)
		if err != nil {
			return nil, errors.New("invalid waiter ID format")
		}
		updateFields["assigned_waiter_id"] = wid
	}

	if len(updateFields) == 0 {
		return table.ToDTO(), nil
	}

	if err := s.tableRepo.Update(ctx, tid, updateFields); err != nil {
		s.logger.Error("Failed to update table", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	updated, _ := s.tableRepo.FindByID(ctx, tid)
	s.logger.Info("Table updated", zap.String("id", id))
	return updated.ToDTO(), nil
}

func (s *tableService) DeleteTable(ctx context.Context, id string) error {
	tid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrTableNotFound
	}

	table, err := s.tableRepo.FindByID(ctx, tid)
	if err != nil || table == nil {
		return ErrTableNotFound
	}

	if table.Status == entities.TableOccupied || table.CurrentOrderID != primitive.NilObjectID {
		return ErrCannotDeleteActive
	}

	if err := s.tableRepo.Delete(ctx, tid); err != nil {
		s.logger.Error("Failed to delete table", zap.String("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("Table deleted", zap.String("id", id))
	return nil
}

func (s *tableService) ListAssignedTables(ctx context.Context, waiterID string) ([]*entities.TableDTO, error) {
	wid, err := primitive.ObjectIDFromHex(waiterID)
	if err != nil {
		return nil, errors.New("invalid waiter ID")
	}

	tables, err := s.tableRepo.FindAssignedToWaiter(ctx, wid)
	if err != nil {
		s.logger.Error("Failed to list assigned tables", zap.String("waiter_id", waiterID), zap.Error(err))
		return nil, err
	}

	dtos := make([]*entities.TableDTO, len(tables))
	for i, t := range tables {
		dtos[i] = t.ToDTO()
	}

	return dtos, nil
}

// Helper
func isValidStatus(status string) bool {
	switch entities.TableStatus(status) {
	case entities.TableAvailable, entities.TableOccupied, entities.TableReserved, entities.TableCleaning:
		return true
	default:
		return false
	}
}
