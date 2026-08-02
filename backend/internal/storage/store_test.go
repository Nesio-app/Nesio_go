package storage

import "testing"

func TestParseRedisOptions(t *testing.T) {
	tests := []struct {
		name     string
		redisURL string
		addr     string
		password string
	}{
		{name: "local address", redisURL: "redis:6379", addr: "redis:6379"},
		{name: "Railway URL", redisURL: "redis://default:secret@redis.railway.internal:6379", addr: "redis.railway.internal:6379", password: "secret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, err := parseRedisOptions(test.redisURL)
			if err != nil {
				t.Fatalf("parseRedisOptions() error = %v", err)
			}
			if options.Addr != test.addr {
				t.Errorf("Addr = %q, want %q", options.Addr, test.addr)
			}
			if options.Password != test.password {
				t.Errorf("Password = %q, want %q", options.Password, test.password)
			}
		})
	}
}
