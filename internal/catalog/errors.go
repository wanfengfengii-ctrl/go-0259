package catalog

import "mycocycle-growbag-transfer-gate/internal/domain"

// newCatalogError builds a stable domain error with the given code, message and
// deterministically sorted reasons.
func newCatalogError(code domain.ErrorCode, message string, reasons ...string) *domain.Error {
	return domain.NewError(code, message, reasons...)
}
