package service

import (
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

var (
	categoryRepo     port.CategoryRepository
	jobRepo          port.JobRepository
	jobLogRepo       port.JobLogRepository
	errorLogRepo     port.ErrorLogRepository
	operationLogRepo port.OperationLogRepository
	friendLinkRepo   port.FriendLinkRepository
	menuRepo         port.MenuRepository
	resourceRepo     port.ResourceRepository
	roleRepo         port.RoleRepository
	tagRepo          port.TagRepository
)

// ConfigureRepositories is called by the composition root during startup.
// Keeping the compatibility registry here lets existing facade handler
// functions retain their signatures while every dependency remains a domain
// port rather than a concrete persistence implementation.
func ConfigureRepositories(
	category port.CategoryRepository,
	job port.JobRepository,
	jobLog port.JobLogRepository,
	errorLog port.ErrorLogRepository,
	operationLog port.OperationLogRepository,
	friendLink port.FriendLinkRepository,
	menu port.MenuRepository,
	resource port.ResourceRepository,
	role port.RoleRepository,
	tag port.TagRepository,
) {
	categoryRepo = category
	jobRepo = job
	jobLogRepo = jobLog
	errorLogRepo = errorLog
	operationLogRepo = operationLog
	friendLinkRepo = friendLink
	menuRepo = menu
	resourceRepo = resource
	roleRepo = role
	tagRepo = tag
}
