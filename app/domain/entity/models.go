// Package entity keeps the historical entity names as compatibility aliases.
//
// Persistence-specific xorm metadata is intentionally not defined here. The
// metadata-bearing rows live in app/infra/persistence/row; new code should use
// the contracts in app/domain/port directly.
package entity

import "benetnasch/app/domain/port"

type TAbout = port.TAbout
type TArticle = port.TArticle
type TArticleTag = port.TArticleTag
type TCategory = port.TCategory
type TComment = port.TComment
type TExceptionLog = port.TExceptionLog
type TFriendLink = port.TFriendLink
type TJob = port.TJob
type TJobLog = port.TJobLog
type TMenu = port.TMenu
type TOperationLog = port.TOperationLog
type TPhoto = port.TPhoto
type TPhotoAlbum = port.TPhotoAlbum
type TResource = port.TResource
type TRole = port.TRole
type TRoleMenu = port.TRoleMenu
type TRoleResource = port.TRoleResource
type TTag = port.TTag
type TTalk = port.TTalk
type TUniqueView = port.TUniqueView
type TUserAuth = port.TUserAuth
type TUserInfo = port.TUserInfo
type TUserRole = port.TUserRole
type TWebsiteConfig = port.TWebsiteConfig
