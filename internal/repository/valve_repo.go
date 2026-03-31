package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type ValveRepo struct {
	pool *pgxpool.Pool
}

func NewValveRepo(pool *pgxpool.Pool) *ValveRepo {
	return &ValveRepo{pool: pool}
}
