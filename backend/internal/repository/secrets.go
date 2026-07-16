package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/secrets"
)

// SecretsRepository handles storing and retrieving encrypted API keys and tokens.
type SecretsRepository interface {
	UpsertSecret(ctx context.Context, businessID uuid.UUID, keyName string, plaintextVal string) (*domain.SystemSecret, error)
	GetSecret(ctx context.Context, businessID uuid.UUID, keyName string) (string, error)
	ListSecretKeys(ctx context.Context, businessID uuid.UUID) ([]domain.SystemSecret, error)
	DeleteSecret(ctx context.Context, businessID uuid.UUID, keyName string) error
}

type postgresSecretsRepository struct {
	q             *db.Queries
	encryptionKey []byte
}

// NewSecretsRepository creates a new SecretsRepository using the provided 32-byte encryption key.
func NewSecretsRepository(q *db.Queries, encryptionKey []byte) (SecretsRepository, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("repository: encryptionKey must be exactly 32 bytes")
	}
	return &postgresSecretsRepository{
		q:             q,
		encryptionKey: encryptionKey,
	}, nil
}

func (r *postgresSecretsRepository) UpsertSecret(ctx context.Context, businessID uuid.UUID, keyName string, plaintextVal string) (*domain.SystemSecret, error) {
	encryptedVal, err := secrets.Encrypt(plaintextVal, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("repository.UpsertSecret encrypt: %w", err)
	}

	row, err := r.q.UpsertSecret(ctx, db.UpsertSecretParams{
		BusinessID:   businessID,
		KeyName:      keyName,
		EncryptedVal: encryptedVal,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.UpsertSecret query: %w", err)
	}

	return &domain.SystemSecret{
		ID:           row.ID,
		BusinessID:   row.BusinessID,
		KeyName:      row.KeyName,
		EncryptedVal: row.EncryptedVal,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *postgresSecretsRepository) GetSecret(ctx context.Context, businessID uuid.UUID, keyName string) (string, error) {
	row, err := r.q.GetSecret(ctx, db.GetSecretParams{
		BusinessID: businessID,
		KeyName:    keyName,
	})
	if err != nil {
		return "", fmt.Errorf("repository.GetSecret query: %w", err)
	}

	plaintext, err := secrets.Decrypt(row.EncryptedVal, r.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("repository.GetSecret decrypt: %w", err)
	}

	return plaintext, nil
}

func (r *postgresSecretsRepository) ListSecretKeys(ctx context.Context, businessID uuid.UUID) ([]domain.SystemSecret, error) {
	rows, err := r.q.ListSecretKeys(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListSecretKeys query: %w", err)
	}

	var results []domain.SystemSecret
	for _, row := range rows {
		results = append(results, domain.SystemSecret{
			ID:         row.ID,
			BusinessID: row.BusinessID,
			KeyName:    row.KeyName,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return results, nil
}

func (r *postgresSecretsRepository) DeleteSecret(ctx context.Context, businessID uuid.UUID, keyName string) error {
	err := r.q.DeleteSecret(ctx, db.DeleteSecretParams{
		BusinessID: businessID,
		KeyName:    keyName,
	})
	if err != nil {
		return fmt.Errorf("repository.DeleteSecret query: %w", err)
	}
	return nil
}
