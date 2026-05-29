// Package types 定义 server 与 cli 共享的请求/响应 DTO。
package types

// HealthResponse 是 /healthz 的返回。
type HealthResponse struct {
	Status string `json:"status"`
}

// MaterialResponse 是上传素材的返回。
type MaterialResponse struct {
	MediaID string `json:"media_id"`
	URL     string `json:"url,omitempty"`
}

// Article 对应微信图文草稿的一篇文章。
type Article struct {
	Title              string `json:"title"`
	Author             string `json:"author,omitempty"`
	Digest             string `json:"digest,omitempty"`
	Content            string `json:"content"`
	ContentSourceURL   string `json:"content_source_url,omitempty"`
	ThumbMediaID       string `json:"thumb_media_id"`
	ShowCoverPic       uint   `json:"show_cover_pic,omitempty"`
	NeedOpenComment    uint   `json:"need_open_comment,omitempty"`
	OnlyFansCanComment uint   `json:"only_fans_can_comment,omitempty"`
}

// DraftRequest 是创建草稿的请求体。
type DraftRequest struct {
	Articles []Article `json:"articles"`
}

// DraftResponse 是创建草稿的返回。
type DraftResponse struct {
	MediaID string `json:"media_id"`
}

// ErrorBody / ErrorResponse 是统一错误信封。
type ErrorBody struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	WechatErrcode int64  `json:"wechat_errcode,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ---- 管理面 DTO ----

type CreateAccountRequest struct {
	Name      string `json:"name"`
	AppID     string `json:"appid"`
	AppSecret string `json:"app_secret"`
}

type UpdateAccountRequest struct {
	Name      *string `json:"name,omitempty"`
	AppID     *string `json:"appid,omitempty"`
	AppSecret *string `json:"app_secret,omitempty"`
}

type AccountResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	AppID string `json:"appid"`
}

type CreateKeyRequest struct {
	Label string `json:"label"`
}

type CreateKeyResponse struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Prefix string `json:"prefix"`
}

type KeyResponse struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"account_id"`
	Prefix    string `json:"prefix"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
	RevokedAt string `json:"revoked_at,omitempty"`
}
