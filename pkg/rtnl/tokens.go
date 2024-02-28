package rtnl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/api/idtoken"
)

var (
	initValidator sync.Once
	validator     *idtoken.Validator
)

func InitValidator(ctx context.Context) (err error) {
	initValidator.Do(func() {
		validator, err = idtoken.NewValidator(ctx)
	})
	return err
}

func (s *Server) ValidateGoogleJWT(credential string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err = InitValidator(ctx); err != nil {
		return err
	}

	var payload *idtoken.Payload
	if payload, err = validator.Validate(ctx, credential, s.conf.GoogleClientID); err != nil {
		return err
	}

	if hd := payload.Claims["hd"].(string); hd != s.conf.AllowedDomain {
		return fmt.Errorf("%s is not an authorized domain", hd)
	}

	return nil
}
