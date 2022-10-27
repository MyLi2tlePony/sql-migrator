package entity

import "time"

type Migration interface {
	GetName() string
	GetStatus() string
	GetVersion() int
	GetStatusChangeTime() time.Time

	SetName(name string)
	SetStatus(status string)
	SetVersion(version int)
	SetStatusChangeTime(statusChangeTime time.Time)
}

type migration struct {
	Name    string
	Version int

	Status           string
	StatusChangeTime time.Time
}

func NewMigration(name, status string, version int, statusChangeTime time.Time) Migration {
	return &migration{
		Name:             name,
		Status:           status,
		Version:          version,
		StatusChangeTime: statusChangeTime,
	}
}

func (m *migration) GetName() string {
	return m.Name
}

func (m *migration) GetStatus() string {
	return m.Status
}

func (m *migration) GetVersion() int {
	return m.Version
}

func (m *migration) GetStatusChangeTime() time.Time {
	return m.StatusChangeTime
}

func (m *migration) SetName(name string) {
	m.Name = name
}

func (m *migration) SetStatus(status string) {
	m.Status = status
}

func (m *migration) SetVersion(version int) {
	m.Version = version
}

func (m *migration) SetStatusChangeTime(statusChangeTime time.Time) {
	m.StatusChangeTime = statusChangeTime
}
