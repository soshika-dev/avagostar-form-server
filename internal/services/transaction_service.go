package services

import (
	"context"

	"avagostar-form-server/internal/models"
	"avagostar-form-server/internal/repo"
)

type TransactionService struct {
	transactions *repo.TransactionRepo
}

func NewTransactionService(transactions *repo.TransactionRepo) *TransactionService {
	return &TransactionService{transactions: transactions}
}

func (s *TransactionService) Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	return s.transactions.Create(ctx, tx)
}

func (s *TransactionService) List(ctx context.Context, filters repo.TransactionFilters) ([]models.Transaction, int64, error) {
	return s.transactions.List(ctx, filters)
}

func (s *TransactionService) Summary(ctx context.Context, filters repo.TransactionFilters) (*repo.TransactionSummary, error) {
	return s.transactions.Summary(ctx, filters)
}

func (s *TransactionService) GetByID(ctx context.Context, id string, createdBy *string) (*models.Transaction, error) {
	return s.transactions.GetByID(ctx, id, createdBy)
}

func (s *TransactionService) Delete(ctx context.Context, id string, createdBy *string) (bool, error) {
	return s.transactions.Delete(ctx, id, createdBy)
}
