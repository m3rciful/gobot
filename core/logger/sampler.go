package logger

import (
	"strconv"
	"strings"
	"sync"
)

type ratioSampler struct {
	mu          sync.Mutex
	numerator   int
	denominator int
	counter     int
}

func newRatioSampler(numerator, denominator int) *ratioSampler {
	s := &ratioSampler{}
	s.Set(numerator, denominator)
	return s
}

// Set configures the sampling ratio using numerator/denominator.
func (s *ratioSampler) Set(numerator, denominator int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if numerator <= 0 || denominator <= 0 {
		s.numerator = 0
		s.denominator = 0
		s.counter = 0
		return
	}
	if numerator > denominator {
		numerator = denominator
	}
	s.numerator = numerator
	s.denominator = denominator
	s.counter = 0
}

// Allow reports whether the current event should pass sampling.
func (s *ratioSampler) Allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.denominator <= 0 || s.numerator <= 0 {
		return true
	}
	s.counter++
	if s.counter > s.denominator {
		s.counter = 1
	}
	return s.counter <= s.numerator
}

func parseRatioSpec(spec string) (int, int) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, 0
	}
	if strings.Contains(spec, "/") {
		parts := strings.SplitN(spec, "/", 2)
		num, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		den, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return num, den
		}
	}
	if v, err := strconv.Atoi(spec); err == nil {
		if v <= 0 {
			return 0, 0
		}
		return 1, v
	}
	return 0, 0
}
