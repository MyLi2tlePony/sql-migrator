package migration

import (
	"time"
)

type migration struct {
	version int
	name    string

	up   string
	down string

	status           string
	statusChangeTime time.Time
}

func (m *migration) GetName() string {
	return m.name
}

func (m *migration) GetStatus() string {
	return m.status
}

func (m *migration) GetVersion() int {
	return m.version
}

func (m *migration) GetStatusChangeTime() time.Time {
	return m.statusChangeTime
}

func (m *migration) SetName(name string) {
	m.name = name
}

func (m *migration) SetStatus(status string) {
	m.status = status
}

func (m *migration) SetVersion(version int) {
	m.version = version
}

func (m *migration) SetStatusChangeTime(statusChangeTime time.Time) {
	m.statusChangeTime = statusChangeTime
}
