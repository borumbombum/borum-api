// Package carddav implements a CardDAV backend for sovereign contact backup.
// It stores vCard data in Turso DB and implements the carddav.Backend interface
// from emersion/go-webdav for compatibility with iOS, Android, and desktop
// CardDAV clients.
package carddav

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// Backend implements carddav.Backend for Turso DB storage.
type Backend struct {
	db *sql.DB
}

// NewBackend creates a new CardDAV backend.
func NewBackend(db *sql.DB) *Backend {
	return &Backend{db: db}
}

// CurrentUserPrincipal returns the current user's principal URL.
func (b *Backend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	return "/carddav/", nil
}

// AddressBookHomeSetPath returns the path to the user's address book home set.
func (b *Backend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	return "/carddav/", nil
}

// ListAddressBooks returns all address books for the user.
func (b *Backend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	userID := userIDFromContext(ctx)
	
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, name, description, created_at FROM address_books WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list address books: %w", err)
	}
	defer rows.Close()

	var books []carddav.AddressBook
	for rows.Next() {
		var book carddav.AddressBook
		var description, createdAt string
		if err := rows.Scan(&book.Path, &book.Name, &description, &createdAt); err != nil {
			return nil, fmt.Errorf("scan address book: %w", err)
		}
		books = append(books, book)
	}
	return books, nil
}

// GetAddressBook returns a specific address book.
func (b *Backend) GetAddressBook(ctx context.Context, path string) (*carddav.AddressBook, error) {
	userID := userIDFromContext(ctx)
	
	var book carddav.AddressBook
	var description, createdAt string
	err := b.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at FROM address_books WHERE id = ? AND user_id = ?`,
		path, userID,
	).Scan(&book.Path, &book.Name, &description, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get address book: %w", err)
	}
	return &book, nil
}

// CreateAddressBook creates a new address book (not supported for sovereign backup).
func (b *Backend) CreateAddressBook(ctx context.Context, book *carddav.AddressBook) error {
	return fmt.Errorf("create address book not supported: use pre-created default address book")
}

// DeleteAddressBook deletes an address book (not supported for sovereign backup).
func (b *Backend) DeleteAddressBook(ctx context.Context, path string) error {
	return fmt.Errorf("delete address book not supported for sovereign backup")
}

// ListAddressObjects returns all contacts in an address book.
func (b *Backend) ListAddressObjects(ctx context.Context, path string, req *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	userID := userIDFromContext(ctx)
	
	rows, err := b.db.QueryContext(ctx,
		`SELECT c.id, c.vcard_data, c.etag, c.updated_at FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE ab.id = ? AND ab.user_id = ? AND c.deleted_at IS NULL`,
		path, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list address objects: %w", err)
	}
	defer rows.Close()

	var objects []carddav.AddressObject
	for rows.Next() {
		var obj carddav.AddressObject
		var vcardData, etag, updatedAt string
		if err := rows.Scan(&obj.Path, &vcardData, &etag, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan address object: %w", err)
		}
		obj.ETag = etag
		objects = append(objects, obj)
	}
	return objects, nil
}

// GetAddressObject returns a specific contact.
func (b *Backend) GetAddressObject(ctx context.Context, path string, req *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	userID := userIDFromContext(ctx)
	
	var obj carddav.AddressObject
	var vcardData, etag, updatedAt string
	err := b.db.QueryRowContext(ctx,
		`SELECT c.id, c.vcard_data, c.etag, c.updated_at FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE c.path = ? AND ab.user_id = ? AND c.deleted_at IS NULL`,
		path, userID,
	).Scan(&obj.Path, &vcardData, &etag, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get address object: %w", err)
	}
	obj.ETag = etag
	return &obj, nil
}

// PutAddressObject creates or updates a contact.
func (b *Backend) PutAddressObject(ctx context.Context, path string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	userID := userIDFromContext(ctx)
	
	// Get the address book ID for the default address book
	var addressBookID string
	err := b.db.QueryRowContext(ctx,
		`SELECT id FROM address_books WHERE user_id = ? LIMIT 1`,
		userID,
	).Scan(&addressBookID)
	if err != nil {
		return nil, fmt.Errorf("get address book: %w", err)
	}

	// Generate ETag from vCard data
	var buf strings.Builder
	enc := vcard.NewEncoder(&buf)
	if err := enc.Encode(card); err != nil {
		return nil, fmt.Errorf("encode vcard: %w", err)
	}
	vcardData := buf.String()
	hash := sha256.Sum256([]byte(vcardData))
	etag := fmt.Sprintf(`"%x"`, hash)

	// Upsert the contact
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = b.db.ExecContext(ctx,
		`INSERT INTO contacts (id, address_book_id, path, vcard_data, etag, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, ?, NULL)
		 ON CONFLICT(address_book_id, path) DO UPDATE SET
		 vcard_data = excluded.vcard_data,
		 etag = excluded.etag,
		 updated_at = excluded.updated_at,
		 deleted_at = NULL`,
		path, addressBookID, path, vcardData, etag, now,
	)
	if err != nil {
		return nil, fmt.Errorf("put address object: %w", err)
	}

	return &carddav.AddressObject{
		Path:    path,
		ETag:    etag,
		Card:    card,
		ModTime: time.Now(),
	}, nil
}

// DeleteAddressObject soft-deletes a contact.
func (b *Backend) DeleteAddressObject(ctx context.Context, path string) error {
	userID := userIDFromContext(ctx)
	
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := b.db.ExecContext(ctx,
		`UPDATE contacts SET deleted_at = ? FROM address_books ab
		 WHERE contacts.address_book_id = ab.id
		 AND contacts.path = ? AND ab.user_id = ? AND contacts.deleted_at IS NULL`,
		now, path, userID,
	)
	if err != nil {
		return fmt.Errorf("delete address object: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// QueryAddressObjects performs a multisearch query (addressbook-multiget).
func (b *Backend) QueryAddressObjects(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	var objects []carddav.AddressObject
	
	if len(query.PropFilters) > 0 {
		// Multisearch: fetch specific UIDs
		for _, filter := range query.PropFilters {
			if filter.Name == "uid" && len(filter.TextMatches) > 0 {
				uid := filter.TextMatches[0].Text
				obj, err := b.GetAddressObject(ctx, uid, &query.DataRequest)
				if err == ErrNotFound {
					continue
				}
				if err != nil {
					return nil, err
				}
				objects = append(objects, *obj)
			}
		}
	} else {
		// Full collection sync
		objs, err := b.ListAddressObjects(ctx, path, &query.DataRequest)
		if err != nil {
			return nil, err
		}
		objects = objs
	}
	
	return objects, nil
}

// userIDFromContext extracts the user ID from the request context.
// For now, this is hardcoded to "default" for single-user sovereign backup.
func userIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return "default"
}

// contextKey is a type for context keys in this package.
type contextKey int

const userIDKey contextKey = 1

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
