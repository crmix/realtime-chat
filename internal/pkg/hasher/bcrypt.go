package hasher

import "golang.org/x/crypto/bcrypt"

// Hasher abstracts password hashing so the service layer doesn't depend on a
// specific crypto library.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type Bcrypt struct {
	cost int
}

func NewBcrypt(cost int) Bcrypt {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return Bcrypt{cost: cost}
}

func (b Bcrypt) Hash(password string) (string, error) {
	out, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (b Bcrypt) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
