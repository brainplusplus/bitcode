package ddd

// ValueObject is compared by its values, not by identity.
type ValueObject interface {
	Equals(other ValueObject) bool
}
