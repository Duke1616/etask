package config

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var sizePattern = regexp.MustCompile(`^(?i)([0-9\.]+)\s*(B|KB|MB|GB|TB)$`)

// ParseBytes 将 "10GB"、"512MB" 等大小格式化字符串解析为字节 int64 数值
func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// 如果本来就是纯数字，直接解析为字节数
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val, nil
	}

	matches := sizePattern.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0, errors.New("invalid size format: " + s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	var multiplier float64
	switch strings.ToUpper(matches[2]) {
	case "B":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, errors.New("unsupported unit: " + matches[2])
	}

	return int64(value * multiplier), nil
}

// StringToBytesSizeHookFunc 支持将带有 KB/MB/GB/TB 单位的字符串自动反序列化为整数字节
func StringToBytesSizeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(int64(0)) && t != reflect.TypeOf(int(0)) {
			return data, nil
		}

		strVal := strings.TrimSpace(data.(string))
		if strVal == "" {
			return data, nil
		}

		// 检查是否符合带大小单位的特征（含英文字母）
		if !hasLetter(strVal) {
			return data, nil
		}

		val, err := ParseBytes(strVal)
		if err != nil {
			return nil, err
		}

		if t == reflect.TypeOf(int(0)) {
			return int(val), nil
		}
		return val, nil
	}
}

func hasLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

// UnmarshalKey 封装了对 viper.UnmarshalKey 的调用，并自动注入了能够解析大小单位 (如 10GB) 以及 time.Duration 的 DecodeHook
func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	defaultOpts := []viper.DecoderConfigOption{
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			StringToBytesSizeHookFunc(),
		)),
	}
	defaultOpts = append(defaultOpts, opts...)
	return viper.UnmarshalKey(key, rawVal, defaultOpts...)
}
