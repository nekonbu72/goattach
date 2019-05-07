package goattach

import "errors"

type ConnInfo struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func (c *ConnInfo) Address() (string, error) {
	if !c.isValid() {
		return "", errors.New("goattach: conn fields error")
	}

	const connectHostPort string = ":"
	return c.Host + connectHostPort + c.Port, nil
}

func (c *ConnInfo) isValid() bool {
	if c.Host == "" || c.Port == "" || c.User == "" || c.Password == "" {
		return false
	}
	return true
}
