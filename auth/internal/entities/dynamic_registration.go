package entities

import (
	"time"

	"github.com/pkg/errors"
)

type DynamicRegistrationParameters struct {
	registrationID string
	contact        string
	provider       string
	code           string
	startedAt      time.Time
	approvePeriod  time.Duration
}

func NewDynamicRegistrationParameters(registerID, code, contact, provider string,
	approvePeriod time.Duration) (*DynamicRegistrationParameters, error) {
	if registerID == "" {
		return nil, errors.Wrap(ErrInvalidParam, "registrationID not set")
	}

	if code == "" {
		return nil, errors.Wrap(ErrInvalidParam, "code not set")
	}

	if code == "" {
		return nil, errors.Wrap(ErrInvalidParam, "code not set")
	}

	if contact == "" {
		return nil, errors.Wrap(ErrInvalidParam, "contact not set")
	}

	if provider == "" {
		return nil, errors.Wrap(ErrInvalidParam, "provider not set")
	}

	if approvePeriod == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "approve period not set")
	}

	return &DynamicRegistrationParameters{
		registrationID: registerID,
		contact:        contact,
		provider:       provider,
		code:           code,
		approvePeriod:  approvePeriod,

		startedAt: time.Now(),
	}, nil
}

func (dr *DynamicRegistrationParameters) RegistrationID() string {
	return dr.registrationID
}

func (dr *DynamicRegistrationParameters) Contact() string {
	return dr.contact
}

func (dr *DynamicRegistrationParameters) Provider() string {
	return dr.provider
}

func (dr *DynamicRegistrationParameters) Code() string {
	return dr.code
}

func (dr *DynamicRegistrationParameters) StartedAt() time.Time {
	return dr.startedAt
}

func (dr *DynamicRegistrationParameters) ApprovePeriod() time.Duration {
	return dr.approvePeriod
}
