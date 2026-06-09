// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package converters

import (
	"errors"
	"strconv"
	"strings"
)

const (
	B  uint64 = 1
	KB        = B << 10
	MB        = KB << 10
	GB        = MB << 10
)

// Converts supported memory formats to bytes and returns the corresponding value.
//
// The available memory specifiers are "mb" or "gb".
func ToBytes(value string) (uint64, error) {
	unit := strings.ToLower(strings.TrimSpace(value[len(value)-2:]))
	val, err := strconv.Atoi(value[:len(value)-2])
	if err != nil {
		return 0, errors.New("incorrect format for value: " + err.Error())
	} else if val <= 0 {
		return 0, errors.New("value must be positive/non-zero")
	}
	switch unit {
	case "mb":
		return uint64(val) * uint64(MB), nil
	case "gb":
		return uint64(val) * uint64(GB), nil
	}
	return 0, errors.New("syntax error for value: " + value)
}
