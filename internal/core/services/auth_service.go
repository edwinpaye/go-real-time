package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/logger"
)

type authService struct {
	userRepo     ports.UserRepository
	tokenService ports.TokenService
	hasher       ports.PasswordHasher
	auditService ports.AuditService
	eventBus     ports.EventBus
	jwtExpiry    time.Duration
}

// NewAuthService instantiates core authentication use-case service.
func NewAuthService(
	userRepo ports.UserRepository,
	tokenService ports.TokenService,
	hasher ports.PasswordHasher,
	auditService ports.AuditService,
	eventBus ports.EventBus,
	jwtExpiryHours int,
) ports.AuthService {
	return &authService{
		userRepo:     userRepo,
		tokenService: tokenService,
		hasher:       hasher,
		auditService: auditService,
		eventBus:     eventBus,
		jwtExpiry:    time.Duration(jwtExpiryHours) * time.Hour,
	}
}

func (s *authService) Register(ctx context.Context, req ports.RegisterRequest, actCtx ports.AuthContext) (*domain.User, error) {
	if _, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
		return nil, errors.New("user with this email already exists")
	}

	hashedPassword, err := s.hasher.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	role := req.Role
	if role == "" {
		role = domain.RoleCashier
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		FullName:     req.FullName,
		PasswordHash: hashedPassword,
		Role:         role,
		IsActive:     true,
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Async audit log
	newVals, _ := json.Marshal(map[string]interface{}{
		"id":        user.ID,
		"email":     user.Email,
		"full_name": user.FullName,
		"role":      user.Role,
	})

	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "users",
		EntityID:   user.ID,
		Action:     domain.AuditActionCreate,
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		NewValues:  string(newVals),
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Real-time notification
	s.eventBus.Publish(domain.ChangeEvent{
		Type:      domain.EventEntityCreated,
		Resource:  domain.ResourceUser,
		EntityID:  user.ID,
		Data:      user,
		ActorID:   actCtx.UserID,
		Timestamp: time.Now().UTC(),
	})

	return user, nil
}

func (s *authService) Login(ctx context.Context, req ports.LoginRequest, actCtx ports.AuthContext) (*ports.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		logger.Warn("Failed login attempt - user not found", logger.Fields{"email": req.Email, "ip": actCtx.IPAddress})
		return nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	if !s.hasher.ComparePassword(user.PasswordHash, req.Password) {
		logger.Warn("Failed login attempt - invalid password", logger.Fields{"email": req.Email, "ip": actCtx.IPAddress})
		return nil, errors.New("invalid email or password")
	}

	token, err := s.tokenService.GenerateToken(user, s.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "auth",
		EntityID:   user.ID,
		Action:     "LOGIN",
		ActorID:    user.ID,
		ActorEmail: user.Email,
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	return &ports.LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *authService) ValidateToken(ctx context.Context, tokenStr string) (*domain.User, error) {
	claims, err := s.tokenService.ValidateToken(tokenStr)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found for token")
	}

	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}

	return user, nil
}
