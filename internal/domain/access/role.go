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
