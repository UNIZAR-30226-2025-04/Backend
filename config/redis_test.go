package config_test

import (
	"Nogler/config"
	"Nogler/redis"
	"testing"
)

func TestConnect_redis(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := config.Connect_redis()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Connect_redis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Connect_redis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Connect_redis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnect_redis(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := config.Connect_redis()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Connect_redis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Connect_redis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Connect_redis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnect_redis(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := config.Connect_redis()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Connect_redis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Connect_redis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Connect_redis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnect_redis(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		want    *redis.RedisClient
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := config.Connect_redis()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Connect_redis() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Connect_redis() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Connect_redis() = %v, want %v", got, tt.want)
			}
		})
	}
}
