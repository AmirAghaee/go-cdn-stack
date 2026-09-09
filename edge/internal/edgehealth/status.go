package edgehealth

import "time"

type Status struct {
	Service   string
	Instance  string
	State     string
	Timestamp time.Time
	Version   string
}
