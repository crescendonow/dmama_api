package service

import (
	"context"
	"fmt"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"
)

type DMAService struct {
	dmaRepo      *repository.DMARepo
	customerRepo *repository.CustomerRepo
	pipeRepo     *repository.PipeRepo
	leakRepo     *repository.LeakpointRepo
}

func NewDMAService(dmaRepo *repository.DMARepo, customerRepo *repository.CustomerRepo, pipeRepo *repository.PipeRepo, leakRepo *repository.LeakpointRepo) *DMAService {
	return &DMAService{
		dmaRepo:      dmaRepo,
		customerRepo: customerRepo,
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
	wkt, err := s.dmaRepo.GetBoundaryRaw(ctx, pwaCode, dmaID)
	if err != nil {
		return nil, fmt.Errorf("fetching DMA boundary: %w", err)
	}
	return s.customerRepo.CountPopulationInDMA(ctx, region, pwaCode, wkt, column)
}

// GetPipeLength fetches DMA boundary then calculates pipe length. (Endpoint 6)
func (s *DMAService) GetPipeLength(ctx context.Context, pwaCode, dmaID string, region int) (*model.DMAPipeLength, error) {
	wkt, err := s.dmaRepo.GetBoundaryRaw(ctx, pwaCode, dmaID)
	if err != nil {
		return nil, fmt.Errorf("fetching DMA boundary: %w", err)
	}
	return s.pipeRepo.SumPipeLengthInDMA(ctx, region, wkt)
}
