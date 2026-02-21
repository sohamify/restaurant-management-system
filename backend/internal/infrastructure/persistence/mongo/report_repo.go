package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

type reportRepo struct {
	orderColl     *mongo.Collection
	paymentColl   *mongo.Collection
	inventoryColl *mongo.Collection
	menuItemColl  *mongo.Collection
}

func NewReportRepository() repositories.ReportRepository {
	db := GetDatabase()
	return &reportRepo{
		orderColl:     db.Collection("orders"),
		paymentColl:   db.Collection("payments"),
		inventoryColl: db.Collection("inventory"),
		menuItemColl:  db.Collection("menu_items"),
	}
}

func (r *reportRepo) AggregateSales(ctx context.Context, filter entities.ReportFilter) (*entities.SalesSummary, error) {
	if filter.StartDate.IsZero() || filter.EndDate.IsZero() {
		return nil, errors.New("start and end dates required")
	}

	match := bson.M{
		"created_at": bson.M{
			"$gte": filter.StartDate,
			"$lte": filter.EndDate,
		},
	}

	groupFormat := "%Y-%m-%d"
	if filter.Period == entities.ReportWeekly {
		groupFormat = "%Y-W%V"
	} else if filter.Period == entities.ReportMonthly {
		groupFormat = "%Y-%m"
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{
					"format": groupFormat,
					"date":   "$created_at",
				},
			},
			"totalRevenue":    bson.M{"$sum": "$total_amount"},
			"orderCount":      bson.M{"$sum": 1},
			"completedOrders": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", string(entities.OrderCompleted)}}, 1, 0}}},
			"cancelledOrders": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", string(entities.OrderCancelled)}}, 1, 0}}},
			"prepTimesMs": bson.M{"$push": bson.M{"$cond": bson.A{
				bson.M{"$and": bson.A{
					bson.M{"$ne": bson.A{"$completed_at", nil}},
					bson.M{"$eq": bson.A{"$status", string(entities.OrderCompleted)}},
				}},
				bson.M{"$subtract": bson.A{"$completed_at", "$created_at"}},
				"$$REMOVE", // Exclude non-completed/nulls from the array for accurate averaging
			}}},
			"hours": bson.M{"$push": bson.M{"$hour": "$created_at"}},
		}}},
		{{Key: "$project", Value: bson.M{
			"period":             "$_id",
			"totalRevenue":       1,
			"orderCount":         1,
			"completedOrders":    1,
			"cancelledOrders":    1,
			"avgOrderValue":      bson.M{"$divide": bson.A{"$totalRevenue", bson.M{"$max": bson.A{"$orderCount", 1}}}},
			"cancellationRate":   bson.M{"$multiply": bson.A{bson.M{"$divide": bson.A{"$cancelledOrders", bson.M{"$max": bson.A{"$orderCount", 1}}}}, 100}},
			"avgPreparationTime": bson.M{"$divide": bson.A{bson.M{"$avg": "$prepTimesMs"}, 60000}},
			"hours":              1,
		}}},
		{{Key: "$sort", Value: bson.M{"period": 1}}},
	}

	cursor, err := r.orderColl.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type aggResult struct {
		Period             string  `bson:"period"`
		TotalRevenue       float64 `bson:"totalRevenue"`
		OrderCount         int     `bson:"orderCount"`
		CompletedOrders    int     `bson:"completedOrders"`
		CancelledOrders    int     `bson:"cancelledOrders"`
		AvgOrderValue      float64 `bson:"avgOrderValue"`
		CancellationRate   float64 `bson:"cancellationRate"`
		AvgPreparationTime float64 `bson:"avgPreparationTime"`
		Hours              []int   `bson:"hours"`
	}

	var results []aggResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &entities.SalesSummary{Period: filter.StartDate.Format("2006-01-02")}, nil
	}

	// Calculate Peak Hour from the hours array
	peakHour := 0
	hourCounts := make(map[int]int)
	maxCount := 0
	for _, h := range results[0].Hours {
		hourCounts[h]++
		if hourCounts[h] > maxCount {
			maxCount = hourCounts[h]
			peakHour = h
		}
	}

	return &entities.SalesSummary{
		Period:             results[0].Period,
		TotalRevenue:       results[0].TotalRevenue,
		OrderCount:         results[0].OrderCount,
		AvgOrderValue:      results[0].AvgOrderValue,
		CompletedOrders:    results[0].CompletedOrders,
		CancelledOrders:    results[0].CancelledOrders,
		CancellationRate:   results[0].CancellationRate,
		AvgPreparationTime: results[0].AvgPreparationTime,
		PeakHour:           peakHour,
	}, nil
}

func (r *reportRepo) AggregateTopItems(ctx context.Context, filter entities.ReportFilter, limit int) ([]*entities.TopItem, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": filter.StartDate, "$lte": filter.EndDate},
			"status":     string(entities.OrderCompleted),
		}}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.M{
			"_id":          "$items.menu_item_id",
			"quantitySold": bson.M{"$sum": "$items.quantity"},
			"revenue":      bson.M{"$sum": bson.M{"$multiply": bson.A{"$items.quantity", "$items.price_at_time"}}},
		}}},
		{{Key: "$sort", Value: bson.M{"revenue": -1}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := r.orderColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var aggItems []struct {
		ID           primitive.ObjectID `bson:"_id"`
		QuantitySold int                `bson:"quantitySold"`
		Revenue      float64            `bson:"revenue"`
	}
	if err := cursor.All(ctx, &aggItems); err != nil {
		return nil, err
	}

	topItems := make([]*entities.TopItem, 0, len(aggItems))
	for _, res := range aggItems {
		var item struct{ Name string }
		_ = r.menuItemColl.FindOne(ctx, bson.M{"_id": res.ID}).Decode(&item)
		name := item.Name
		if name == "" {
			name = "Unknown"
		}

		topItems = append(topItems, &entities.TopItem{
			MenuItemID:   res.ID.Hex(),
			Name:         name,
			QuantitySold: res.QuantitySold,
			Revenue:      res.Revenue,
		})
	}
	return topItems, nil
}

func (r *reportRepo) GetInventoryStatus(ctx context.Context) (*entities.InventoryStatus, error) {
	cursor, err := r.inventoryColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []entities.InventoryItem
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	statusItems := make([]*entities.InventoryStatusItem, 0, len(items))
	lowCount := 0
	for _, item := range items {
		isLow := item.Quantity < item.LowStockThreshold
		if isLow {
			lowCount++
		}
		statusItems = append(statusItems, &entities.InventoryStatusItem{
			ID:                item.ID.Hex(),
			Name:              item.Name,
			Unit:              item.Unit,
			CurrentQuantity:   item.Quantity,
			LowStockThreshold: item.LowStockThreshold,
			IsLowStock:        isLow,
		})
	}

	return &entities.InventoryStatus{
		Items:         statusItems,
		LowStockCount: lowCount,
	}, nil
}

func (r *reportRepo) AggregatePaymentBreakdown(ctx context.Context, filter entities.ReportFilter) ([]*entities.PaymentBreakdown, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": filter.StartDate, "$lte": filter.EndDate},
			"status":     string(entities.PaymentCompleted),
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":         "$method",
			"count":       bson.M{"$sum": 1},
			"totalAmount": bson.M{"$sum": "$amount"},
		}}},
		{{Key: "$sort", Value: bson.M{"totalAmount": -1}}},
	}

	cursor, err := r.paymentColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		Method      string  `bson:"_id"`
		Count       int     `bson:"count"`
		TotalAmount float64 `bson:"totalAmount"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	breakdowns := make([]*entities.PaymentBreakdown, len(results))
	for i, res := range results {
		breakdowns[i] = &entities.PaymentBreakdown{
			Method:      res.Method,
			Count:       res.Count,
			TotalAmount: res.TotalAmount,
		}
	}
	return breakdowns, nil
}

func (r *reportRepo) AggregateRefundSummary(ctx context.Context, filter entities.ReportFilter) (*entities.RefundSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": filter.StartDate, "$lte": filter.EndDate},
			"status":     string(entities.PaymentRefunded),
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"totalRefunded": bson.M{"$sum": "$amount"},
			"refundCount":   bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.paymentColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var aggRes struct {
		TotalRefunded float64 `bson:"totalRefunded"`
		RefundCount   int     `bson:"refundCount"`
	}
	if cursor.Next(ctx) {
		_ = cursor.Decode(&aggRes)
	}

	summary := &entities.RefundSummary{
		TotalRefunded: aggRes.TotalRefunded,
		RefundCount:   aggRes.RefundCount,
	}

	// Calculate refund rate based on revenue
	sales, err := r.AggregateSales(ctx, filter)
	if err == nil && sales.TotalRevenue > 0 {
		summary.RefundRate = (aggRes.TotalRefunded / sales.TotalRevenue) * 100
	}

	return summary, nil
}

func (r *reportRepo) GetPastRevenueData(ctx context.Context, periods int) ([]float64, error) {
	// Optimization: Single aggregation query instead of a loop
	startDate := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -periods)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startDate},
			"status":     string(entities.OrderCompleted),
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$created_at"},
			},
			"revenue": bson.M{"$sum": "$total_amount"},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := r.orderColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		Revenue float64 `bson:"revenue"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	revenues := make([]float64, len(results))
	for i, res := range results {
		revenues[i] = res.Revenue
	}
	return revenues, nil
}
