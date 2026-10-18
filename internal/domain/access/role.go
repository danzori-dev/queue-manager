package access

type Role uint8

const (
	RoleOutsider Role = iota
	RoleAdmin
	RoleOwner
)

func (r Role) AtLeast(minRole Role) bool {
	return r >= minRole
}

func (r Role) String() string {
	switch r {
	case RoleOwner:
		return "owner"
	case RoleAdmin:
		return "admin"
	case RoleOutsider:
		return "outsider"
	default:
		return "unknown"
	}
}
