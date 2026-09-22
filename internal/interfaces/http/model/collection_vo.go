package model

type CollectionSaveVO struct {
	Title       string `json:"title" form:"title"`
	Description string `json:"description" form:"description"`
	Visibility  string `json:"visibility" form:"visibility"`
}

type CollectionItemVO struct {
	Note string `json:"note" form:"note"`
}

type CollectionOrderVO struct {
	ArticleIDs []int `json:"articleIds" form:"articleIds"`
}
