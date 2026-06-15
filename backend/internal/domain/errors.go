// Package domain — Sentinel errors.
//
// Centralised error values for the domain layer. Service and handler
// layers map these to appropriate gRPC status codes or HTTP responses.
package domain

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInsufficientStock is returned when an order line requests more units
	// than are currently available in the product's stock_qty.
	ErrInsufficientStock = errors.New("insufficient stock")

	// ErrInvalidSKU is returned when a submitted SKU does not exist in
	// the active tenant's product catalog.
	ErrInvalidSKU = errors.New("invalid SKU")

	// ErrInvalidOrder is returned when an OrderRequest fails structural
	// validation (e.g., empty items list, zero quantity).
	ErrInvalidOrder = errors.New("invalid order")

	// ErrUnauthorised is returned when the caller lacks permission.
	ErrUnauthorised = errors.New("unauthorised")
)
