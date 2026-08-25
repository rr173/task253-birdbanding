package model

import "errors"

// 领域错误集合。业务包与 store 统一返回这些错误，HTTP 层据此映射状态码。
var (
	ErrInvalidRingFormat        = errors.New("环号格式错误")
	ErrRecaptureTimeReversed    = errors.New("重捕时间倒退")
	ErrLocationPrecisionMissing = errors.New("地点精度缺失")
	ErrIdentityConflict         = errors.New("环号身份冲突")
	ErrFrozenImmutable          = errors.New("冻结版本不可修改")
	ErrInvalidTransition        = errors.New("状态流转非法")
	ErrNotFound                 = errors.New("记录不存在")
	ErrDuplicate                = errors.New("重复导入")
	ErrInvalidArgument          = errors.New("参数非法")
	ErrRateOverLimit            = errors.New("迁徙边速度超限")
)

// IsNotFound 判断错误是否为「记录不存在」。
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
