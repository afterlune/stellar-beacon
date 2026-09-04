package row

import "benetnasch/app/domain/port"

// The conversions below are deliberately explicit at the persistence edge.
// Domain and row structs have the same field layout, but keeping the
// conversion here makes it impossible for an xorm-tagged type to escape into
// application code by accident.

func ToAbout(value port.TAbout) TAbout   { return TAbout(value) }
func FromAbout(value TAbout) port.TAbout { return port.TAbout(value) }

func ToArticle(value port.TArticle) TArticle   { return TArticle(value) }
func FromArticle(value TArticle) port.TArticle { return port.TArticle(value) }
func FromArticles(values []TArticle) []port.TArticle {
	result := make([]port.TArticle, len(values))
	for index, value := range values {
		result[index] = FromArticle(value)
	}
	return result
}

func ToArticleTag(value port.TArticleTag) TArticleTag   { return TArticleTag(value) }
func FromArticleTag(value TArticleTag) port.TArticleTag { return port.TArticleTag(value) }

func ToCategory(value port.TCategory) TCategory   { return TCategory(value) }
func FromCategory(value TCategory) port.TCategory { return port.TCategory(value) }
func FromCategories(values []TCategory) []port.TCategory {
	result := make([]port.TCategory, len(values))
	for index, value := range values {
		result[index] = FromCategory(value)
	}
	return result
}

func ToComment(value port.TComment) TComment   { return TComment(value) }
func FromComment(value TComment) port.TComment { return port.TComment(value) }
func FromComments(values []TComment) []port.TComment {
	result := make([]port.TComment, len(values))
	for index, value := range values {
		result[index] = FromComment(value)
	}
	return result
}

func ToExceptionLog(value port.TExceptionLog) TExceptionLog   { return TExceptionLog(value) }
func FromExceptionLog(value TExceptionLog) port.TExceptionLog { return port.TExceptionLog(value) }
func FromExceptionLogs(values []TExceptionLog) []port.TExceptionLog {
	result := make([]port.TExceptionLog, len(values))
	for index, value := range values {
		result[index] = FromExceptionLog(value)
	}
	return result
}

func ToFriendLink(value port.TFriendLink) TFriendLink   { return TFriendLink(value) }
func FromFriendLink(value TFriendLink) port.TFriendLink { return port.TFriendLink(value) }
func FromFriendLinks(values []TFriendLink) []port.TFriendLink {
	result := make([]port.TFriendLink, len(values))
	for index, value := range values {
		result[index] = FromFriendLink(value)
	}
	return result
}

func ToJob(value port.TJob) TJob   { return TJob(value) }
func FromJob(value TJob) port.TJob { return port.TJob(value) }
func FromJobs(values []TJob) []port.TJob {
	result := make([]port.TJob, len(values))
	for index, value := range values {
		result[index] = FromJob(value)
	}
	return result
}

func ToJobLog(value port.TJobLog) TJobLog   { return TJobLog(value) }
func FromJobLog(value TJobLog) port.TJobLog { return port.TJobLog(value) }
func FromJobLogs(values []TJobLog) []port.TJobLog {
	result := make([]port.TJobLog, len(values))
	for index, value := range values {
		result[index] = FromJobLog(value)
	}
	return result
}

func ToMenu(value port.TMenu) TMenu   { return TMenu(value) }
func FromMenu(value TMenu) port.TMenu { return port.TMenu(value) }
func FromMenus(values []TMenu) []port.TMenu {
	result := make([]port.TMenu, len(values))
	for index, value := range values {
		result[index] = FromMenu(value)
	}
	return result
}

func ToOperationLog(value port.TOperationLog) TOperationLog   { return TOperationLog(value) }
func FromOperationLog(value TOperationLog) port.TOperationLog { return port.TOperationLog(value) }
func FromOperationLogs(values []TOperationLog) []port.TOperationLog {
	result := make([]port.TOperationLog, len(values))
	for index, value := range values {
		result[index] = FromOperationLog(value)
	}
	return result
}

func ToPhoto(value port.TPhoto) TPhoto   { return TPhoto(value) }
func FromPhoto(value TPhoto) port.TPhoto { return port.TPhoto(value) }
func FromPhotos(values []TPhoto) []port.TPhoto {
	result := make([]port.TPhoto, len(values))
	for index, value := range values {
		result[index] = FromPhoto(value)
	}
	return result
}

func ToPhotoAlbum(value port.TPhotoAlbum) TPhotoAlbum   { return TPhotoAlbum(value) }
func FromPhotoAlbum(value TPhotoAlbum) port.TPhotoAlbum { return port.TPhotoAlbum(value) }
func FromPhotoAlbums(values []TPhotoAlbum) []port.TPhotoAlbum {
	result := make([]port.TPhotoAlbum, len(values))
	for index, value := range values {
		result[index] = FromPhotoAlbum(value)
	}
	return result
}

func ToResource(value port.TResource) TResource   { return TResource(value) }
func FromResource(value TResource) port.TResource { return port.TResource(value) }
func FromResources(values []TResource) []port.TResource {
	result := make([]port.TResource, len(values))
	for index, value := range values {
		result[index] = FromResource(value)
	}
	return result
}

func ToRole(value port.TRole) TRole   { return TRole(value) }
func FromRole(value TRole) port.TRole { return port.TRole(value) }
func FromRoles(values []TRole) []port.TRole {
	result := make([]port.TRole, len(values))
	for index, value := range values {
		result[index] = FromRole(value)
	}
	return result
}

func ToRoleMenu(value port.TRoleMenu) TRoleMenu             { return TRoleMenu(value) }
func ToRoleResource(value port.TRoleResource) TRoleResource { return TRoleResource(value) }
func ToTag(value port.TTag) TTag                            { return TTag(value) }
func FromTag(value TTag) port.TTag                          { return port.TTag(value) }
func FromTags(values []TTag) []port.TTag {
	result := make([]port.TTag, len(values))
	for index, value := range values {
		result[index] = FromTag(value)
	}
	return result
}

func ToTalk(value port.TTalk) TTalk   { return TTalk(value) }
func FromTalk(value TTalk) port.TTalk { return port.TTalk(value) }
func FromTalks(values []TTalk) []port.TTalk {
	result := make([]port.TTalk, len(values))
	for index, value := range values {
		result[index] = FromTalk(value)
	}
	return result
}

func ToUniqueView(value port.TUniqueView) TUniqueView   { return TUniqueView(value) }
func FromUniqueView(value TUniqueView) port.TUniqueView { return port.TUniqueView(value) }

func ToUserAuth(value port.TUserAuth) TUserAuth   { return TUserAuth(value) }
func FromUserAuth(value TUserAuth) port.TUserAuth { return port.TUserAuth(value) }

func ToUserInfo(value port.TUserInfo) TUserInfo   { return TUserInfo(value) }
func FromUserInfo(value TUserInfo) port.TUserInfo { return port.TUserInfo(value) }

func ToUserRole(value port.TUserRole) TUserRole                  { return TUserRole(value) }
func ToWebsiteConfig(value port.TWebsiteConfig) TWebsiteConfig   { return TWebsiteConfig(value) }
func FromWebsiteConfig(value TWebsiteConfig) port.TWebsiteConfig { return port.TWebsiteConfig(value) }
