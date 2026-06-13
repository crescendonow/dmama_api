package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"golang.org/x/sync/singleflight"
)

const statsCacheTTL = 24 * time.Hour

type DMAService struct {
	dmaRepo      *repository.DMARepo
	customerRepo *repository.CustomerRepo
	meterRepo    *repository.MeterRepo
	pipeRepo     *repository.PipeRepo
	leakRepo     *repository.LeakpointRepo
	statsCache   sync.Map
	statsGroup   singleflight.Group
}

type statsCacheEntry struct {
	value     *model.DMAStats
	expiresAt time.Time
}

func NewDMAService(dmaRepo *repository.DMARepo, customerRepo *repository.CustomerRepo, meterRepo *repository.MeterRepo, pipeRepo *repository.PipeRepo, leakRepo *repository.LeakpointRepo) *DMAService {
	return &DMAService{
		dmaRepo:      dmaRepo,
		customerRepo: customerRepo,
		meterRepo:    meterRepo,
		pipeRepo:     pipeRepo,
		leakRepo:     leakRepo,
	}
}

// GetUsage fetches DMA boundary then calculates usage. (Endpoint 4)
func (s *DMAService) GetUsage(ctx context.Context, pwaCode, dmaID, column string, region int) (*model.DMAUsage, error) {
	wkt, err := s.dmaRepo.GetBoundaryRaw(ctx, pwaCode, dmaID)
	if err != nil {
		return nil, fmt.Errorf("fetching DMA boundary: %w", err)
	}
	return s.customerRepo.SumUsageInDMA(ctx, region, pwaCode, wkt, column)
}

// GetPopulation fetches DMA boundary then counts population. (Endpoint 5)
func (s *DMAService) GetPopulation(ctx context.Context, pwaCode, dmaID, column string, region int) (*model.DMAPopulation, error) {
	return s.customerRepo.CountPopulationByDMA(ctx, region, pwaCode, dmaID, column)
}

// GetStats returns merged usage and population statistics for a DMA. (Endpoint stats)
func (s *DMAService) GetStats(ctx context.Context, pwaCode, dmaID, column string, region int) (*model.DMAStats, error) {
	key := fmt.Sprintf("%d:%s:%s:%s", region, pwaCode, dmaID, column)
	now := time.Now()
	if cached, ok := s.statsCache.Load(key); ok {
		entry := cached.(statsCacheEntry)
		if now.Before(entry.expiresAt) {
			return prepareDMAStatsResponse(entry.value, pwaCode, dmaID, column), nil
		}
		s.statsCache.Delete(key)
	}

	value, err, _ := s.statsGroup.Do(key, func() (interface{}, error) {
		result, err := s.customerRepo.GetStatsInDMA(ctx, region, pwaCode, dmaID, column)
		if err != nil || result == nil {
			return result, err
		}
		result = prepareDMAStatsResponse(result, pwaCode, dmaID, column)
		s.statsCache.Store(key, statsCacheEntry{
			value:     cloneDMAStats(result),
			expiresAt: time.Now().Add(statsCacheTTL),
		})
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	return prepareDMAStatsResponse(value.(*model.DMAStats), pwaCode, dmaID, column), nil
}

// GetDailyMeterCount counts active meters within a DMA. The column parameter is accepted by the handler for stats payload compatibility but is not used here.
func (s *DMAService) GetDailyMeterCount(ctx context.Context, pwaCode, dmaID string, region int) (*model.DMADailyMeterCount, error) {
	return s.meterRepo.CountDailyMetersInDMA(ctx, region, pwaCode, dmaID)
}

// GetPipeLength fetches DMA boundary then calculates pipe length. (Endpoint 6)
func (s *DMAService) GetPipeLength(ctx context.Context, pwaCode, dmaID string, region int) (*model.DMAPipeLength, error) {
	wkt, err := s.dmaRepo.GetBoundaryRaw(ctx, pwaCode, dmaID)
	if err != nil {
		return nil, fmt.Errorf("fetching DMA boundary: %w", err)
	}
	return s.pipeRepo.SumPipeLengthInDMA(ctx, region, wkt)
}

// ResolveStatsColumn maps stats request params to a safe database column.
func ResolveStatsColumn(yearStr, monthStr, column string, now time.Time) (string, error) {
	if yearStr != "" || monthStr != "" {
		if yearStr == "" || monthStr == "" {
			return "", fmt.Errorf("year and month must be provided together")
		}

		year, err := strconv.Atoi(yearStr)
		if err != nil {
			return "", fmt.Errorf("year must be a number")
		}
		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			return "", fmt.Errorf("month must be 1-12")
		}

		diffMonth := (now.Year()*12 + int(now.Month())) - (year*12 + month)
		if now.Day() < 20 {
			diffMonth--
		}
		if diffMonth < 0 {
			return "", fmt.Errorf("requested month is newer than the latest completed billing cycle")
		}
		if diffMonth > 12 {
			diffMonth = 12
		}
		if diffMonth == 0 {
			return "prswtusg", nil
		}
		return fmt.Sprintf("lstwtusg%d", diffMonth), nil
	}

	if column == "" {
		column = "prswtusg"
	}
	if err := repository.ValidateColumn(column); err != nil {
		return "", err
	}
	return column, nil
}

func cloneDMAStats(stats *model.DMAStats) *model.DMAStats {
	if stats == nil {
		return nil
	}
	copied := *stats
	return &copied
}

func prepareDMAStatsResponse(stats *model.DMAStats, pwaCode, dmaID, column string) *model.DMAStats {
	copied := cloneDMAStats(stats)
	if copied == nil {
		return nil
	}
	copied.PwaCode = pwaCode
	copied.DmaID = dmaID
	copied.Column = column
	return copied
}
