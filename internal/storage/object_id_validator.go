package storage

import (
	"fmt"
	"github.com/google/uuid"
)

func validateObjectId(id string) error {
	if id == "" {
		return fmt.Errorf("object ID is empty")
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf(
			"invalid object ID %q: %w",
			id,
			err,
		)
	}

	if parsedID.String() != id {
		return fmt.Errorf(
			"object ID %q is not in canonical UUID format",
			id,
		)
	}

	return nil
}
