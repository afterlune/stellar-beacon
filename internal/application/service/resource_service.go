package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type ResourceService interface {
	ListResources(ctx context.Context, keywords string) ([]entity.TResource, error)
	DeleteResource(ctx context.Context, id int) error
	SaveOrUpdateResource(ctx context.Context, resource entity.TResource) error
	ListResourceOptions(ctx context.Context) ([]entity.TResource, error)
}

type MyResourceService struct{ repo port.ResourceRepository }

func NewResourceService(repo port.ResourceRepository) *MyResourceService {
	return &MyResourceService{repo: repo}
}

func (r *MyResourceService) resourceRepository() port.ResourceRepository {
	if r.repo != nil {
		return r.repo
	}
	return resourceRepo
}

func (r *MyResourceService) ListResources(ctx context.Context, keywords string) ([]entity.TResource, error) {
	return r.resourceRepository().List(ctx, keywords)
}

func (r *MyResourceService) DeleteResource(ctx context.Context, id int) error {
	return r.resourceRepository().Delete(ctx, id)
}

func (r *MyResourceService) SaveOrUpdateResource(ctx context.Context, resource entity.TResource) error {
	return r.resourceRepository().SaveOrUpdate(ctx, resource)
}

func (r *MyResourceService) ListResourceOptions(ctx context.Context) ([]entity.TResource, error) {
	return r.resourceRepository().ListOptions(ctx)
}
