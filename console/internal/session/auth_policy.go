package session

import "fmt"

// AuthPolicy selects shared or per-project credentials. Its zero value is global
// auth, so every value has one of the two supported meanings.
type AuthPolicy struct{ project bool }

func GlobalAuthPolicy() AuthPolicy  { return AuthPolicy{} }
func ProjectAuthPolicy() AuthPolicy { return AuthPolicy{project: true} }

// ParseAuthPolicy parses the persisted representation at the storage boundary.
func ParseAuthPolicy(raw string) (AuthPolicy, error) {
	switch raw {
	case "global":
		return GlobalAuthPolicy(), nil
	case "project":
		return ProjectAuthPolicy(), nil
	default:
		return AuthPolicy{}, fmt.Errorf("unsupported auth policy: %q", raw)
	}
}

func (p AuthPolicy) IsProject() bool { return p.project }

func (p AuthPolicy) String() string {
	if p.project {
		return "project"
	}
	return "global"
}
