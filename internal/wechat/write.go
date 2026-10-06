package wechat

import (
	"fmt"
	"io"
	"net/url"

	"github.com/azure1489/mp-helper/internal/types"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"github.com/silenceper/wechat/v2/util"
)

// UploadBodyImage uses the SDK's reader helper, not permanent material storage.
// v2.1.13 ImageUpload only accepts a path, so build its uploadimg request here.
func (m *Manager) UploadBodyImage(appID, appSecret, filename string, r io.Reader) (string, error) {
	token, err := m.get(appID, appSecret).GetAccessToken()
	if err != nil {
		return "", err
	}
	uri := "https://api.weixin.qq.com/cgi-bin/media/uploadimg?access_token=" + url.QueryEscape(token)
	response, err := util.PostFileFromReader("media", "", filename, uri, r)
	if err != nil {
		return "", err
	}
	var result struct {
		util.CommonError
		URL string `json:"url"`
	}
	if err = util.DecodeWithError(response, &result, "UploadBodyImage"); err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", fmt.Errorf("uploadimg returned an empty url")
	}
	return result.URL, nil
}

// UpdateDraft writes exactly the requested ID/index once. No create fallback/retry.
func (m *Manager) UpdateDraft(appID, appSecret, mediaID string, index int, a types.Article) error {
	return m.get(appID, appSecret).GetDraft().UpdateDraft(&draft.Article{
		Title: a.Title, Author: a.Author, Digest: a.Digest, Content: a.Content,
		ContentSourceURL: a.ContentSourceURL, ThumbMediaID: a.ThumbMediaID,
		ShowCoverPic: a.ShowCoverPic, NeedOpenComment: a.NeedOpenComment,
		OnlyFansCanComment: a.OnlyFansCanComment,
	}, mediaID, uint(index))
}
