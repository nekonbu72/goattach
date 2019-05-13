package mailg

import "errors"

type ConnInfo struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func (c *ConnInfo) address() (string, error) {
	if !c.isValid() {
		return "", errors.New("mailg: conninfo fields error")
	}

	const delimiter string = ":"
	return c.Host + delimiter + c.Port, nil
}

func (c *ConnInfo) isValid() bool {
	if c.Host == "" || c.Port == "" || c.User == "" || c.Password == "" {
		return false
	}
	return true
}
