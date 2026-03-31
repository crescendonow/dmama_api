package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeterRepo struct {
	pool *pgxpool.Pool
}

func NewMeterRepo(pool *pgxpool.Pool) *MeterRepo {
	return &MeterRepo{pool: pool}
}
