package auth

import "github.com/sayuri567/tool/component/baidupan"

type AuthClient struct {
	baiduPan *baidupan.BaiduPan
}

func GetClient(key string) *AuthClient {
	return &AuthClient{
		baiduPan: baidupan.GetBaiduPan(key),
	}
}

// func (this *AuthClient)
