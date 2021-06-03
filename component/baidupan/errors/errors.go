package errors

import "errors"

var (
	Errs = map[int]error{
		2:     errors.New("参数错误"),
		-6:    errors.New("身份验证失败"),
		31034: errors.New("命中接口频控"),
		42000: errors.New("访问过于频繁"),
		42001: errors.New("rand校验失败"),
		42999: errors.New("功能下线"),
		9100:  errors.New("一级封禁"),
		9200:  errors.New("二级封禁"),
		9300:  errors.New("三级封禁"),
		9400:  errors.New("四级封禁"),
		9500:  errors.New("五级封禁"),
	}
)
