package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

type SearchHandler struct {
	pb.UnimplementedPersonSearchServiceServer
	search *usecase.PersonSearchUseCase
}

func NewSearchHandler(search *usecase.PersonSearchUseCase) *SearchHandler {
	return &SearchHandler{search: search}
}

func (h *SearchHandler) SearchPersons(ctx context.Context, req *pb.SearchPersonsRequest) (*pb.SearchPersonsResponse, error) {
	sort := usecase.PersonSearchSort("invalid")

	switch req.GetSort() {
	case pb.PersonSearchSort_PERSON_SEARCH_SORT_UNSPECIFIED, pb.PersonSearchSort_PERSON_SEARCH_SORT_RELEVANCE:
		sort = usecase.PersonSearchSortRelevance
	case pb.PersonSearchSort_PERSON_SEARCH_SORT_ADDED_AT:
		sort = usecase.PersonSearchSortAddedAt
	}

	result, err := h.search.SearchPersons(ctx, usecase.SearchPersonsInput{
		Criteria: personCriteria(req.GetQuery(), req.GetRoles()),
		Sort:     sort, Limit: int(req.GetLimit()), Offset: req.GetOffset(),
		IncludeWorksCount: req.GetIncludeWorksCount(),
	})
	if err != nil {
		return nil, personSearchError(err)
	}

	people := make([]*pb.PersonSuggestion, 0, len(result.Persons))
	for _, match := range result.Persons {
		roles := make([]pb.SearchPersonRole, 0, len(match.Roles))
		for _, role := range match.Roles {
			switch role {
			case usecase.SearchPersonRoleActor:
				roles = append(roles, pb.SearchPersonRole_SEARCH_PERSON_ROLE_ACTOR)
			case usecase.SearchPersonRoleDirector:
				roles = append(roles, pb.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR)
			}
		}

		people = append(people, &pb.PersonSuggestion{
			Id: match.Person.ID, Name: match.Person.Name, PhotoUrl: match.Person.PhotoURL,
			Roles: roles, WorksCount: match.WorksCount,
		})
	}

	return &pb.SearchPersonsResponse{Persons: people, Total: result.Total}, nil
}

func (h *SearchHandler) CountPersons(ctx context.Context, req *pb.CountPersonsRequest) (*pb.CountPersonsResponse, error) {
	result, err := h.search.CountPersons(ctx, usecase.CountPersonsInput{Criteria: personCriteria(req.GetQuery(), req.GetRoles())})
	if err != nil {
		return nil, personSearchError(err)
	}

	return &pb.CountPersonsResponse{Total: result.Total}, nil
}

func personCriteria(query string, filter *pb.PersonRoleFilter) usecase.PersonSearchCriteria {
	mode := searchquery.FilterMode("invalid")

	switch filter.GetMode() {
	case pb.PersonRoleFilterMode_PERSON_ROLE_FILTER_MODE_UNSPECIFIED, pb.PersonRoleFilterMode_PERSON_ROLE_FILTER_MODE_ALL:
		mode = searchquery.FilterAll
	case pb.PersonRoleFilterMode_PERSON_ROLE_FILTER_MODE_ANY:
		mode = searchquery.FilterAny
	}

	return usecase.PersonSearchCriteria{Query: query, Roles: searchquery.Filter[usecase.SearchPersonRole]{
		Include: personRolesFromProto(filter.GetInclude()),
		Exclude: personRolesFromProto(filter.GetExclude()),
		Mode:    mode,
	}}
}

func personRolesFromProto(roles []pb.SearchPersonRole) []usecase.SearchPersonRole {
	values := make([]usecase.SearchPersonRole, 0, len(roles))
	for _, role := range roles {
		value := usecase.SearchPersonRole("invalid")

		switch role {
		case pb.SearchPersonRole_SEARCH_PERSON_ROLE_ACTOR:
			value = usecase.SearchPersonRoleActor
		case pb.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR:
			value = usecase.SearchPersonRoleDirector
		}

		values = append(values, value)
	}

	return values
}

func personSearchError(err error) error {
	switch {
	case errors.Is(err, searchquery.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return toGRPCError(err)
	}
}
