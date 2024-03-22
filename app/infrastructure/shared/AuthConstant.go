package shared

import "time"

const (
	TWENTY_MINUTES = 20 * time.Minute

	EXPIRE_TIME = 7 * 24 * time.Hour

	TOKEN_HEADER = "Authorization"

	TOKEN_PREFIX = "Bearer "

	SECRET = "红白"

	ACCESS_LIMIT = 60
)
