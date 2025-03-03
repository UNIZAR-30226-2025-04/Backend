package redis_test

import (
	"Nogler/services/redis"
	"testing"
)

func TestInitRedis(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		Addr    string
		DB      int
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := redis.InitRedis(tt.Addr, tt.DB)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("InitRedis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("InitRedis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("InitRedis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitRedis(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		Addr    string
		DB      int
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := redis.InitRedis(tt.Addr, tt.DB)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("InitRedis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("InitRedis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("InitRedis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitRedis(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		Addr    string
		DB      int
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := redis.InitRedis(tt.Addr, tt.DB)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("InitRedis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("InitRedis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("InitRedis() = %v, want %v", got, tt.want)
			}
		})
	}
}
