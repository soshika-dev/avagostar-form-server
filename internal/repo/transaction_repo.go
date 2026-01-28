package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"avagostar-form-server/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepo struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

type TransactionFilters struct {
	Search    string
	DateFrom  *time.Time
	DateTo    *time.Time
	Currency  string
	MinAmount *float64
	Month     *int
	SortBy    string
	SortDir   string
	Page      int
	PerPage   int
	CreatedBy *string
}

type TransactionSummary struct {
	TotalAmount float64
	AvgAmount   float64
	Count       int64
	Monthly     map[string]float64
	ByCurrency  map[string]float64
}

func NewTransactionRepo(pool *pgxpool.Pool, timeout time.Duration) *TransactionRepo {
	return &TransactionRepo{pool: pool, timeout: timeout}
}

func (r *TransactionRepo) Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	row := r.pool.QueryRow(ctx, `
		INSERT INTO transactions (
			created_by_user_id, receiver_type, receiver_name, receiver_id,
			payer_type, payer_name, payer_id, payment_method, currency,
			amount, description, datetime_utc, timezone
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at
	`,
		tx.CreatedByUserID,
		tx.ReceiverType,
		tx.ReceiverName,
		tx.ReceiverID,
		tx.PayerType,
		tx.PayerName,
		tx.PayerID,
		tx.PaymentMethod,
		tx.Currency,
		tx.Amount,
		tx.Description,
		tx.DatetimeUTC,
		tx.Timezone,
	)

	if err := row.Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	return tx, nil
}

func (r *TransactionRepo) GetByID(ctx context.Context, id string, createdBy *string) (*models.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query := `
		SELECT id, created_by_user_id, receiver_type, receiver_name, receiver_id,
		payer_type, payer_name, payer_id, payment_method, currency, amount,
		description, datetime_utc, timezone, created_at, updated_at
		FROM transactions
		WHERE id = $1`
	args := []any{id}
	if createdBy != nil {
		query += " AND created_by_user_id = $2"
		args = append(args, *createdBy)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	var tx models.Transaction
	if err := row.Scan(
		&tx.ID,
		&tx.CreatedByUserID,
		&tx.ReceiverType,
		&tx.ReceiverName,
		&tx.ReceiverID,
		&tx.PayerType,
		&tx.PayerName,
		&tx.PayerID,
		&tx.PaymentMethod,
		&tx.Currency,
		&tx.Amount,
		&tx.Description,
		&tx.DatetimeUTC,
		&tx.Timezone,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return &tx, nil
}

func (r *TransactionRepo) Delete(ctx context.Context, id string, createdBy *string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query := "DELETE FROM transactions WHERE id = $1"
	args := []any{id}
	if createdBy != nil {
		query += " AND created_by_user_id = $2"
		args = append(args, *createdBy)
	}

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("delete transaction: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *TransactionRepo) List(ctx context.Context, filters TransactionFilters) ([]models.Transaction, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	whereSQL, args := buildTransactionFilters(filters)

	sortColumn := mapSortColumn(filters.SortBy)
	sortDir := "ASC"
	if strings.ToLower(filters.SortDir) == "desc" {
		sortDir = "DESC"
	}

	limit := filters.PerPage
	if limit <= 0 {
		limit = 10
	}
	offset := (filters.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, created_by_user_id, receiver_type, receiver_name, receiver_id,
		payer_type, payer_name, payer_id, payment_method, currency, amount,
		description, datetime_utc, timezone, created_at, updated_at
		FROM transactions
		%s
		ORDER BY %s %s
		LIMIT %d OFFSET %d
	`, whereSQL, sortColumn, sortDir, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var results []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.CreatedByUserID,
			&tx.ReceiverType,
			&tx.ReceiverName,
			&tx.ReceiverID,
			&tx.PayerType,
			&tx.PayerName,
			&tx.PayerID,
			&tx.PaymentMethod,
			&tx.Currency,
			&tx.Amount,
			&tx.Description,
			&tx.DatetimeUTC,
			&tx.Timezone,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		results = append(results, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate transactions: %w", err)
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM transactions %s`, whereSQL)
	row := r.pool.QueryRow(ctx, countQuery, args...)
	var total int64
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	return results, total, nil
}

func (r *TransactionRepo) Summary(ctx context.Context, filters TransactionFilters) (*TransactionSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	whereSQL, args := buildTransactionFilters(filters)

	kpiQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount),0), COALESCE(AVG(amount),0), COUNT(*)
		FROM transactions
		%s
	`, whereSQL)
	row := r.pool.QueryRow(ctx, kpiQuery, args...)
	var summary TransactionSummary
	if err := row.Scan(&summary.TotalAmount, &summary.AvgAmount, &summary.Count); err != nil {
		return nil, fmt.Errorf("summary kpis: %w", err)
	}

	monthlyQuery := fmt.Sprintf(`
		SELECT TO_CHAR(DATE_TRUNC('month', datetime_utc), 'MM') AS month,
		COALESCE(SUM(amount),0)
		FROM transactions
		%s
		GROUP BY 1
		ORDER BY 1
	`, whereSQL)
	monthlyRows, err := r.pool.Query(ctx, monthlyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("summary monthly: %w", err)
	}
	defer monthlyRows.Close()

	summary.Monthly = make(map[string]float64)
	for monthlyRows.Next() {
		var month string
		var amount float64
		if err := monthlyRows.Scan(&month, &amount); err != nil {
			return nil, fmt.Errorf("scan monthly: %w", err)
		}
		summary.Monthly[month] = amount
	}
	if err := monthlyRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monthly: %w", err)
	}

	currencyQuery := fmt.Sprintf(`
		SELECT currency, COALESCE(SUM(amount),0)
		FROM transactions
		%s
		GROUP BY currency
	`, whereSQL)
	currencyRows, err := r.pool.Query(ctx, currencyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("summary currency: %w", err)
	}
	defer currencyRows.Close()

	summary.ByCurrency = make(map[string]float64)
	for currencyRows.Next() {
		var currency string
		var amount float64
		if err := currencyRows.Scan(&currency, &amount); err != nil {
			return nil, fmt.Errorf("scan currency: %w", err)
		}
		summary.ByCurrency[currency] = amount
	}
	if err := currencyRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate currency: %w", err)
	}

	return &summary, nil
}

func buildTransactionFilters(filters TransactionFilters) (string, []any) {
	clauses := []string{"WHERE 1=1"}
	args := []any{}
	index := 1

	if filters.CreatedBy != nil {
		clauses = append(clauses, fmt.Sprintf("AND created_by_user_id = $%d", index))
		args = append(args, *filters.CreatedBy)
		index++
	}

	if filters.Search != "" {
		clauses = append(clauses, fmt.Sprintf("AND (receiver_name ILIKE $%d OR payer_name ILIKE $%d)", index, index))
		args = append(args, "%"+filters.Search+"%")
		index++
	}

	if filters.DateFrom != nil {
		clauses = append(clauses, fmt.Sprintf("AND datetime_utc >= $%d", index))
		args = append(args, *filters.DateFrom)
		index++
	}

	if filters.DateTo != nil {
		clauses = append(clauses, fmt.Sprintf("AND datetime_utc < $%d", index))
		args = append(args, *filters.DateTo)
		index++
	}

	if filters.Currency != "" {
		clauses = append(clauses, fmt.Sprintf("AND currency = $%d", index))
		args = append(args, filters.Currency)
		index++
	}

	if filters.MinAmount != nil {
		clauses = append(clauses, fmt.Sprintf("AND amount >= $%d", index))
		args = append(args, *filters.MinAmount)
		index++
	}

	if filters.Month != nil {
		clauses = append(clauses, fmt.Sprintf("AND EXTRACT(MONTH FROM datetime_utc) = $%d", index))
		args = append(args, *filters.Month)
		index++
	}

	return strings.Join(clauses, "\n"), args
}

func mapSortColumn(sortBy string) string {
	switch strings.ToLower(sortBy) {
	case "receiver":
		return "receiver_name"
	case "payer":
		return "payer_name"
	case "amount":
		return "amount"
	case "currency":
		return "currency"
	case "date":
		return "datetime_utc"
	default:
		return "datetime_utc"
	}
}
