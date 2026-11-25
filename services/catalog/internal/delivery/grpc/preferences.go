package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) GetPreferences(ctx context.Context, req *pb.GetPreferencesRequest) (*pb.GetPreferencesResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	preferences, err := h.preferences.GetPreferences(ctx, domain.UserID(req.GetUserId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	var view *string

	if preferences.CatalogView != nil {
		value := string(*preferences.CatalogView)
		view = &value
	}

	return &pb.GetPreferencesResponse{
		Preferences: &pb.Preferences{
			SidebarCollapsed: preferences.SidebarCollapsed,
			CatalogView:      view,
		},
	}, nil
}

func (h *CatalogHandler) SavePreference(ctx context.Context, req *pb.SavePreferenceRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	var update domain.PreferenceUpdate

	switch value := req.GetUpdate().(type) {
	case *pb.SavePreferenceRequest_SidebarCollapsed:
		if value == nil {
			return nil, toGRPCError(domain.ErrInvalidPreferences)
		}

		collapsed := value.SidebarCollapsed
		update.SidebarCollapsed = &collapsed
	case *pb.SavePreferenceRequest_CatalogView:
		if value == nil {
			return nil, toGRPCError(domain.ErrInvalidPreferences)
		}

		view := domain.CatalogView(value.CatalogView)
		update.CatalogView = &view
	default:
		return nil, toGRPCError(domain.ErrInvalidPreferences)
	}

	if err := h.preferences.SavePreference(ctx, domain.UserID(req.GetUserId()), update); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}
