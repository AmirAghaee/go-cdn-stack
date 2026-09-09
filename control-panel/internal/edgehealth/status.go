package edgehealth

import "time"

type Status struct {
	ID        string
	Service   string
	Instance  string
	Status    string
	Timestamp time.Time
	Version   string
}
