package style

import "fmt"

type ErrAlreadyExists struct {
	Style Style
	List  *List
}

func (err ErrAlreadyExists) Error() string {
	return fmt.Sprintf("style %v already exists in list", err.Style.Name)
}
