package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
)

func mapDomainToPBSession(s domain.Session) *pb.Session {
	return &pb.Session{
		Id:        s.ID.String(),
		UserId:    int64(s.UserID),
		Role:      s.Role,
		ExpiresAt: timestamppb.New(s.ExpiresAt),
	}
}

type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	usecase usecase.AuthUseCase
}

func NewAuthHandler(auc usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		usecase: auc,
	}
}

func (h *AuthHandler) IssueSession(ctx context.Context, req *pb.IssueSessionRequest) (*pb.IssueSessionResponse, error) {
	session, err := h.usecase.IssueSession(ctx, req.UserId, req.Role)
	if err != nil {
		return nil, grpcutil.ToGRPCError(err)
	}

	return &pb.IssueSessionResponse{
		Session: mapDomainToPBSession(*session),
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	session, err := h.usecase.Login(ctx, req.GetEmail(), req.GetPassword())
	if errors.Is(err, utils.ErrInvalidEmail) || errors.Is(err, utils.ErrInvalidPassword) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if errors.Is(err, domain.ErrInvalidCredentials) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	if err != nil {
		return nil, grpcutil.ToGRPCError(err)
	}

	return &pb.LoginResponse{
		Session: mapDomainToPBSession(*session),
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session id format")
	}

	err = h.usecase.Logout(ctx, sessionID)
	if err != nil {
		return nil, grpcutil.ToGRPCError(err)
	}

	return &pb.LogoutResponse{}, nil
}

func (h *AuthHandler) CheckSession(ctx context.Context, req *pb.CheckSessionRequest) (*pb.CheckSessionResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session id format")
	}

	info, err := h.usecase.CheckSession(ctx, sessionID)
	if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) {
		return nil, status.Error(codes.Unauthenticated, "session not found or expired")
	}

	if err != nil {
		return nil, grpcutil.ToGRPCError(err)
	}

	return &pb.CheckSessionResponse{
		UserId: info.UserID,
		Role:   info.Role,
	}, nil
}

func (h *AuthHandler) RefreshSession(ctx context.Context, req *pb.RefreshSessionRequest) (*pb.RefreshSessionResponse, error) {
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session id format")
	}

	session, err := h.usecase.RefreshSession(ctx, sessionID)
	if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) {
		return nil, status.Error(codes.Unauthenticated, domain.ErrSessionNotFound.Error())
	}

	if err != nil {
		return nil, grpcutil.ToGRPCError(err)
	}

	if session == nil {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.RefreshSessionResponse{Session: mapDomainToPBSession(*session)}, nil
}
