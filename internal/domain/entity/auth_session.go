package entity

import "time"

// AuthSession is the persisted authenticated-user snapshot shared by the
// token middleware and authentication infrastructure. Its JSON field names
// are part of the existing Redis session representation.
type AuthSession struct {
	Id                     int       `json:"id"`
	UserInfoId             int       `json:"userInfoId"`
	Email                  string    `json:"email"`
	LoginType              int       `json:"loginType"`
	Username               string    `json:"username"`
	Password               string    `json:"password"`
	Roles                  []string  `json:"roles"`
	Handle                 string    `json:"handle"`
	Nickname               string    `json:"nickname"`
	Avatar                 string    `json:"avatar"`
	Intro                  string    `json:"intro"`
	Website                string    `json:"website"`
	IsDisable              int       `json:"isDisable"`
	IpAddress              string    `json:"ipAddress"`
	IpSource               string    `json:"ipSource"`
	IsSubscribe            int       `json:"isSubscribe"`
	NotifyComment          int       `json:"notifyComment"`
	NotifyInteraction      int       `json:"notifyInteraction"`
	NotifyTopic            int       `json:"notifyTopic"`
	NotifyCollection       int       `json:"notifyCollection"`
	NotifyStudioActivation int       `json:"notifyStudioActivation"`
	Browser                string    `json:"browser"`
	Os                     string    `json:"os"`
	ExpireTime             time.Time `json:"expireTime"`
	LastLoginTime          time.Time `json:"lastLoginTime"`
}
