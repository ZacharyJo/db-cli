package redisdb

import (
	"strings"
	"testing"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil value", nil, "(nil)"},
		{"string", "hello", "hello"},
		{"integer", int64(42), "(integer) 42"},
		{"empty array", []interface{}{}, "(empty array)"},
		{"string array", []interface{}{"a", "b"}, "1) a\n2) b"},
		{"nested nil", []interface{}{nil}, "1) (nil)"},
		{"empty map interface", map[interface{}]interface{}{}, "(empty map)"},
		{"empty map string", map[string]interface{}{}, "(empty map)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.input)
			if got != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConnectDefaultAddrs(t *testing.T) {
	// 验证空 Addrs 时默认填充 127.0.0.1:6379
	// Connect 会尝试真实连接并失败，我们通过检查 error 中是否包含默认地址来间接验证。
	_, err := Connect(Options{})
	if err == nil {
		t.Skip("unexpected: redis available at 127.0.0.1:6379")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:6379") {
		t.Errorf("expected error to mention default addr 127.0.0.1:6379, got: %v", err)
	}
}

func TestConnectDefaultMode(t *testing.T) {
	// 验证空 Mode 时默认为 ModeSingle（通过检查错误中只有单地址，而非尝试 sentinel/cluster）
	_, err := Connect(Options{Addrs: []string{"127.0.0.1:19999"}})
	if err == nil {
		t.Skip("unexpected: redis available at 127.0.0.1:19999")
	}
	// 单机模式失败时错误中包含 "redis connect"
	if !strings.Contains(err.Error(), "redis connect") {
		t.Errorf("expected connect error, got: %v", err)
	}
}
